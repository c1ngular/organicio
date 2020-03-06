#include <stdio.h>
#include <stdlib.h>

static void die(char *str) {
    fprintf(stderr, "%s\n", str);
    exit(1);
}

static AVStream *add_audio_stream(AVFormatContext *oc, enum AVCodecID codec_id)
{
    AVCodecContext *c;
    AVCodec *encoder = avcodec_find_encoder(codec_id);
    AVStream *st = avformat_new_stream(oc, encoder);

    if (!st) die("av_new_stream");

    c = st->codec;
    c->codec_id = codec_id;
    c->codec_type = AVMEDIA_TYPE_AUDIO;

    /* put sample parameters */
    c->bit_rate = 64000;
    c->sample_rate = 44100;
    c->channels = 2;
    c->sample_fmt = encoder->sample_fmts[0];
    c->channel_layout = AV_CH_LAYOUT_STEREO;

    // some formats want stream headers to be separate
    if(oc->oformat->flags & AVFMT_GLOBALHEADER)
        c->flags |= CODEC_FLAG_GLOBAL_HEADER;

    return st;
}

static void open_audio(AVFormatContext *oc, AVStream *st)
{
    AVCodecContext *c = st->codec;
    AVCodec *codec;

    /* find the audio encoder */
    codec = avcodec_find_encoder(c->codec_id);
    if (!codec) die("avcodec_find_encoder");

    /* open it */
    AVDictionary *dict = NULL;
    av_dict_set(&dict, "strict", "+experimental", 0);
    int res = avcodec_open2(c, codec, &dict);
    if (res < 0) die("avcodec_open");
}

int main(int argc, char *argv[]) {
    av_register_all();

    if (argc != 3) {
        fprintf(stderr, "%s <in> <out>\n", argv[0]);
        exit(1);
    }

    // Allocate and init re-usable frames
    AVCodecContext *fileCodecContext, *audioCodecContext;
    AVFormatContext *formatContext, *outContext;
    AVStream *audioStream;
    SwrContext *swrContext;
    int streamId;

    // input file
    const char *file = argv[1];
    int res = avformat_open_input(&formatContext, file, NULL, NULL);
    if (res != 0) die("avformat_open_input");
    res = avformat_find_stream_info(formatContext, NULL);
    if (res < 0) die("avformat_find_stream_info");
    AVCodec *codec;
    res = av_find_best_stream(formatContext, AVMEDIA_TYPE_AUDIO, -1, -1, &codec, 0);
    if (res < 0) die("av_find_best_stream");
    streamId = res;
    fileCodecContext = avcodec_alloc_context3(codec);
    avcodec_copy_context(fileCodecContext, formatContext->streams[streamId]->codec);
    res = avcodec_open2(fileCodecContext, codec, NULL);
    if (res < 0) die("avcodec_open2");

    // output file
    const char *outfile = argv[2];
    AVOutputFormat *fmt = fmt = av_guess_format(NULL, outfile, NULL);
    if (!fmt) die("av_guess_format");
    outContext = avformat_alloc_context();
    outContext->oformat = fmt;
    audioStream = add_audio_stream(outContext, fmt->audio_codec);
    open_audio(outContext, audioStream);
    res = avio_open2(&outContext->pb, outfile, AVIO_FLAG_WRITE, NULL, NULL);
    if (res < 0) die("url_fopen");
    avformat_write_header(outContext, NULL);
    audioCodecContext = audioStream->codec;

    // resampling
    swrContext = swr_alloc();
    av_opt_set_channel_layout(swrContext, "in_channel_layout",  fileCodecContext->channel_layout, 0);
    av_opt_set_channel_layout(swrContext, "out_channel_layout", audioCodecContext->channel_layout, 0);
    av_opt_set_int(swrContext, "in_sample_rate", fileCodecContext->sample_rate, 0);
    av_opt_set_int(swrContext, "out_sample_rate", audioCodecContext->sample_rate, 0);
    av_opt_set_sample_fmt(swrContext, "in_sample_fmt", fileCodecContext->sample_fmt, 0);
    av_opt_set_sample_fmt(swrContext, "out_sample_fmt", audioCodecContext->sample_fmt, 0);
    res = swr_init(swrContext);
    if (res < 0) die("swr_init");

    AVFrame *audioFrameDecoded = av_frame_alloc();
    if (!audioFrameDecoded)
        die("Could not allocate audio frame");

    audioFrameDecoded->format = fileCodecContext->sample_fmt;
    audioFrameDecoded->channel_layout = fileCodecContext->channel_layout;
    audioFrameDecoded->channels = fileCodecContext->channels;
    audioFrameDecoded->sample_rate = fileCodecContext->sample_rate;

    AVFrame *audioFrameConverted = av_frame_alloc();
    if (!audioFrameConverted) die("Could not allocate audio frame");

    audioFrameConverted->nb_samples = audioCodecContext->frame_size;
    audioFrameConverted->format = audioCodecContext->sample_fmt;
    audioFrameConverted->channel_layout = audioCodecContext->channel_layout;
    audioFrameConverted->channels = audioCodecContext->channels;
    audioFrameConverted->sample_rate = audioCodecContext->sample_rate;

    AVPacket inPacket;
    av_init_packet(&inPacket);
    inPacket.data = NULL;
    inPacket.size = 0;

    int frameFinished = 0;

    while (av_read_frame(formatContext, &inPacket) >= 0) {
        if (inPacket.stream_index == streamId) {
            int len = avcodec_decode_audio4(fileCodecContext, audioFrameDecoded, &frameFinished, &inPacket);

            if (frameFinished) {

                // Convert

                uint8_t *convertedData=NULL;

                if (av_samples_alloc(&convertedData,
                             NULL,
                             audioCodecContext->channels,
                             audioFrameConverted->nb_samples,
                             audioCodecContext->sample_fmt, 0) < 0)
                    die("Could not allocate samples");

                int outSamples = swr_convert(swrContext, NULL, 0,
                             //&convertedData,
                             //audioFrameConverted->nb_samples,
                             (const uint8_t **)audioFrameDecoded->data,
                             audioFrameDecoded->nb_samples);
                if (outSamples < 0) die("Could not convert");

                for (;;) {
                     outSamples = swr_get_out_samples(swrContext, 0);
                     if (outSamples < audioCodecContext->frame_size * audioCodecContext->channels) break; // see comments, thanks to @dajuric for fixing this

                     outSamples = swr_convert(swrContext,
                                              &convertedData,
                                              audioFrameConverted->nb_samples, NULL, 0);

                     size_t buffer_size = av_samples_get_buffer_size(NULL,
                                    audioCodecContext->channels,
                                    audioFrameConverted->nb_samples,
                                    audioCodecContext->sample_fmt,
                                    0);
                    if (buffer_size < 0) die("Invalid buffer size");

                    if (avcodec_fill_audio_frame(audioFrameConverted,
                             audioCodecContext->channels,
                             audioCodecContext->sample_fmt,
                             convertedData,
                             buffer_size,
                             0) < 0)
                        die("Could not fill frame");

                    AVPacket outPacket;
                    av_init_packet(&outPacket);
                    outPacket.data = NULL;
                    outPacket.size = 0;

                    if (avcodec_encode_audio2(audioCodecContext, &outPacket, audioFrameConverted, &frameFinished) < 0)
                        die("Error encoding audio frame");

                    if (frameFinished) {
                        outPacket.stream_index = audioStream->index;

                        if (av_interleaved_write_frame(outContext, &outPacket) != 0)
                            die("Error while writing audio frame");

                        av_free_packet(&outPacket);
                    }
                }
            }
        }
    }

    swr_close(swrContext);
    swr_free(&swrContext);
    av_frame_free(&audioFrameConverted);
    av_frame_free(&audioFrameDecoded);
    av_free_packet(&inPacket);
    av_write_trailer(outContext);
    avio_close(outContext->pb);
    avcodec_close(fileCodecContext);
    avcodec_free_context(&fileCodecContext);
    avformat_close_input(&formatContext);

    return 0;
}