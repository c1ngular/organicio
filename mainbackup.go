package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"sync"
	"syscall"
	"github.com/3d0c/gmf"
	"path/filepath"
)

const (

	HW_VIDEO_CODEC_NAME_H264_MMAL    = "h264_mmal"
	HW_VIDEO_CODEC_NAME_H264_OMX     = "h264_omx"
	HW_VIDEO_CODEC_NAME_H264_V4L2M2M = "h264_v4l2m2m"

	VIDEO_CODEC_ENCODER_NAME_X264    = "libx264"
	VIDEO_CODEC_DECODER_NAME_X264    = "h264"

	AUDIO_DECODE_CODEC_NAME_AAC = "aac"
	AUDIO_ENCODE_CODEC_NAME_AAC = "aac"
	AUDIO_AAC_OUTPUT_CHANNELS=2
	AUDIO_AAC_OUTPUT_SAMPLE_RATE=44100
	AUDIO_DECODE_CODEC_NAME_MP3 = "mp3"

	STREAM_RESOLUTION_HIGHT  = 21
	STREAM_RESOLUTION_MEDIUM = 28
	STREAM_RESOLUTION_LOW    = 34

	STREAM_MAX_BITRATE     = "1M"
	STREAM_BUFSIZE         = "2M"
	STREAM_AUDIO_BITRATE   = "64K"
	STREAM_VIDEO_BITRATE   = "900K"
	STREAM_VIDEO_FRAMERATE = 24

)

var (

	AUDIO_AAC_OUTPUT_SAMPLE_FORMAT int32 =gmf.AV_SAMPLE_FMT_FLTP
	VIDEO_OUTPUT_PIX_FORMAT int32 =gmf.AV_PIX_FMT_YUV420P
	VIDEO_OUTPUT_264_PROFILE int =gmf.FF_PROFILE_H264_BASELINE
)

const (

	STREAMER_UUID     = "sdfrweqrewqr0943rasdfds"
	STREAMER_SECRET   = "FDQER345435435"
	STREAMER_PUSH_URL = "/Users/s1ngular/GoWork/src/github.com/organicio/tesdtt.mp4"
	WATERMARK_IMG_URL="/Users/s1ngular/GoWork/src/github.com/organicio/watermark.png"
	WATERMARK_POSITION_BOTTOM_RIGHT="overlay=main_w-overlay_w-10:main_h-overlay_h-10"
	WATERMARK_POSITION_BOTTOM_LEFT="overlay=10:main_h-overlay_h-10"
	WATERMARK_POSITION_TOP_RIGHT="overlay=main_w-overlay_w-10:10"
	WATERMARK_POSITION_TOP_LEFT="overlay=10:10"
	MP3S_FOLDER_PATH="/Users/s1ngular/GoWork/src/github.com/organicio/mp3s"
)

type StreamInfo struct {
	Schema   string `json:"schema"`
	Vhost    string `json:"vhost"`
	AppName  string `json:"app"`
	StreamId string `json:"stream"`
	UID      string
}

type Mstream struct {
	minfo         StreamInfo
	inctx         *gmf.FmtCtx
	inastream     *gmf.Stream
	invstream     *gmf.Stream
	inaCodec      *gmf.Codec
	invCodec      *gmf.Codec
	inaDecodecCtx *gmf.CodecCtx
	invDecodecCtx *gmf.CodecCtx
}

type Streamer struct {
	mstreams            map[string]*Mstream
	currentStreamingUID string

	outctx        *gmf.FmtCtx
	outvstream    *gmf.Stream
	outastream    *gmf.Stream
	outvCodec     *gmf.Codec
	outaCodec     *gmf.Codec
	outvEncodeCtx *gmf.CodecCtx
	outaEncodeCtx *gmf.CodecCtx
	outvOptions   []gmf.Option
	outaOptions   []gmf.Option

	watermarkInCtx *gmf.FmtCtx
	watermarkInstream *gmf.Stream
	watermarkFilter *gmf.Filter
	waterMarkImgPacket *gmf.Packet
	waterMarkImgFrame *gmf.Frame

	mp3InCtxs []*gmf.FmtCtx
	mp3InStreams []*gmf.Stream
	mp3EncodeCtxs []*gmf.CodecCtx
	currentStreamingMp3Index int

	mux           sync.Mutex
}


func (s *Streamer) initMp3s() error{

	var err error
	var inctx *gmf.FmtCtx
	var mp3Stream *gmf.Stream
	var mp3Decodec *gmf.Codec
	var mp3DecodecCtx *gmf.CodecCtx
	//var mp3Encodec *gmf.Codec
	//var mp3EncodeCtx *gmf.CodecCtx

	mp3s, err := ioutil.ReadDir(MP3S_FOLDER_PATH)
	if err != nil {
		return fmt.Errorf("Error reading - %s\n", err)
	}

	for _, mp3 := range mp3s {

		ext := filepath.Ext(mp3.Name())
		if ext != ".mp3" {
			fmt.Printf("skipping %s, ext: '%s'\n", mp3.Name(), ext)
			continue
		}

		inctx, err = gmf.NewInputCtx(filepath.Join(MP3S_FOLDER_PATH, mp3.Name()))
		if err != nil {
			inctx.Free()
			fmt.Printf("Error creating mp3 input context for %s - %s\n", mp3.Name(),err)
		}else{
			s.mp3InCtxs=append(s.mp3InCtxs,inctx)
		}

		mp3Stream, err = inctx.GetBestStream(gmf.AVMEDIA_TYPE_AUDIO)
		if err != nil {
			mp3Stream.Free()
			log.Printf("No audio stream found in mp3 : '%s'\n", mp3.Name())
			return fmt.Errorf("No audio stream found in mp3 : '%s'", mp3.Name())
		}else{
			s.mp3InStreams=append(s.mp3InStreams,mp3Stream)
		}

		/* mp3 stream decode context set up */
		mp3Decodec, err = gmf.FindDecoder(AUDIO_DECODE_CODEC_NAME_MP3)
		if err != nil {
			log.Printf("coud not fine audio stream decoder for '%s'\n", mp3.Name())
			return fmt.Errorf("coud not find mp3 stream decoder for '%s'", mp3.Name())
		}

		if mp3DecodecCtx = gmf.NewCodecCtx(mp3Decodec); mp3DecodecCtx == nil {
			return fmt.Errorf("unable to create mp3 decode codec context for %s", mp3.Name())
		}

		if err = mp3Stream.GetCodecPar().ToContext(mp3DecodecCtx); err != nil {
			return fmt.Errorf("Failed to copy mp3 decoder parameters to input decoder context  for %s",mp3.Name())
		}

		if err = mp3DecodecCtx.Open(nil); err != nil {
			return fmt.Errorf("Failed to open decoder for mp3 stream  for %s", mp3.Name())
		}
	}	

	return err
}


func (s *Streamer) getMp3sFrames() ([]*gmf.Frame , error){

	var err error
	var pkt *gmf.Packet
	var frame *gmf.Frame
	var frames []*gmf.Frame
	var errInt int

	
	resampleMp3Options := []*gmf.Option{
		{Key: "in_channel_layout", Val: s.mp3InStreams[s.currentStreamingMp3Index].CodecCtx().ChannelLayout()},
		{Key: "out_channel_layout", Val: s.outaEncodeCtx.ChannelLayout()},
		{Key: "in_sample_rate", Val: s.mp3InStreams[s.currentStreamingMp3Index].CodecCtx().SampleRate()},
		{Key: "out_sample_rate", Val: s.outaEncodeCtx.SampleRate()},
		{Key: "in_sample_fmt", Val: gmf.SampleFormat(s.mp3InStreams[s.currentStreamingMp3Index].CodecCtx().SampleFmt())},
		{Key: "out_sample_fmt", Val: gmf.SampleFormat(s.outaEncodeCtx.SampleFmt())},
	}

	if s.outastream.SwrCtx == nil {

		if s.outastream.SwrCtx, err = gmf.NewSwrCtx(resampleMp3Options, s.outaEncodeCtx.Channels(), s.outaEncodeCtx.SampleFmt()); err != nil {
			panic(err)
		}
		s.outastream.AvFifo = gmf.NewAVAudioFifo(s.mp3InStreams[s.currentStreamingMp3Index].CodecCtx().SampleFmt(), s.mp3InStreams[s.currentStreamingMp3Index].CodecCtx().Channels(), 1024)

	}else{
		s.outastream.AvFifo.Free()
		s.outastream.SwrCtx.Free()
		if s.outastream.SwrCtx, err = gmf.NewSwrCtx(resampleMp3Options, s.outaEncodeCtx.Channels(), s.outaEncodeCtx.SampleFmt()); err != nil {
			panic(err)
		}
		s.outastream.AvFifo = gmf.NewAVAudioFifo(s.mp3InStreams[s.currentStreamingMp3Index].CodecCtx().SampleFmt(), s.mp3InStreams[s.currentStreamingMp3Index].CodecCtx().Channels(), 1024)
	}

	var actx=s.mp3InCtxs[s.currentStreamingMp3Index]

	pkt, err = actx.GetNextPacket()
	if err != nil && err != io.EOF {
		if pkt != nil {
			pkt.Free()
		}
		return frames, fmt.Errorf("error getting next packet - %s", err)
		
	} else if err != nil && pkt == nil {

		actx.SeekFile(s.mp3InStreams[s.currentStreamingMp3Index], 0,int64(actx.Duration()), 0)
		fmt.Printf(" reaching end of current mp3 stream \n")
		if s.currentStreamingMp3Index == len(s.mp3InCtxs) - 1{
			s.currentStreamingMp3Index=0
		}else{
			s.currentStreamingMp3Index ++
		}
		
		return s.getMp3sFrames()

	}

	streamIdx := pkt.StreamIndex()
	if streamIdx == s.mp3InStreams[s.currentStreamingMp3Index].Index(){

		frame, errInt = s.mp3InStreams[s.currentStreamingMp3Index].CodecCtx().Decode2(pkt)

		if errInt < 0 && gmf.AvErrno(errInt) == syscall.EAGAIN {
			return s.getMp3sFrames()
		} else if errInt == gmf.AVERROR_EOF {
			return frames,fmt.Errorf("EOF in mp3 audio Decode2, handle it\n")
		} else if errInt < 0 {
			return frames ,fmt.Errorf("Unexpected error mp3 audio - %s\n", gmf.AvError(errInt))
		}

		frames = gmf.DefaultResampler(s.outastream, []*gmf.Frame{frame}, false)		

		return frames , nil

	}else{
		return s.getMp3sFrames()
	}
	
}

func (s *Streamer) clearUpMp3sRes(){

	for i,  _ := range s.mp3InCtxs {

		s.mp3InStreams[i].Free()
		s.mp3InCtxs[i].Free()

	}
}

func (s *Streamer) initWaterMark(suid string)error{

	var err error
	var filename=WATERMARK_IMG_URL
	s.watermarkInCtx, err = gmf.NewInputCtx(filename)
	if(err != nil){
		s.watermarkInCtx.Free()
		return fmt.Errorf("failed to create watermark input context for %s",filename)
	}

	s.watermarkInstream,err=s.watermarkInCtx.GetBestStream(gmf.AVMEDIA_TYPE_VIDEO)
	if(err != nil){
		s.watermarkInstream.Free()
		return fmt.Errorf("failed to create watermark input stream for %s",filename)
	}

	
	s.watermarkFilter, err = gmf.NewFilter(WATERMARK_POSITION_TOP_RIGHT, []*gmf.Stream{s.mstreams[suid].invstream, s.watermarkInstream}, s.outvstream, []*gmf.Option{})
	if err != nil {
		s.watermarkFilter.Release()
		return err
	}	

	err=s.getWatermarkIMGPacket()
	return err
}

/* seems static picture packet number  and frame number is alway 1 ? to be figured out */

func (s *Streamer) getWatermarkIMGPacket() error {
	var err error
	var pkt *gmf.Packet

	
	pkt, err =s.watermarkInCtx.GetNextPacket()
	if err != nil && err != io.EOF {
		if pkt != nil {
			pkt.Free()
		}
		return err
	} else if err != nil && pkt == nil {
		return err
	}

	s.waterMarkImgPacket=pkt
	s.waterMarkImgFrame, _=s.watermarkInstream.CodecCtx().Decode2(s.waterMarkImgPacket)
	
	return err
}

func (s *Streamer) clearUpWaterMarkRes(){

	if s.waterMarkImgFrame != nil{
		s.waterMarkImgFrame.Free()
	}
	if s.waterMarkImgPacket != nil{
		s.waterMarkImgPacket.Free()
	}	
	s.watermarkFilter.Release()
	s.watermarkInstream.Free()
	s.watermarkInCtx.Free()
}

func (s *Streamer) addStream(mInfo StreamInfo) error {

	var err error

	m := &Mstream{minfo: mInfo}

	if !s.streamExisted(m.minfo.UID) {
		m.inctx, err = gmf.NewInputCtx(m.minfo.UID)
		if err != nil {
			m.inctx.Free()
			fmt.Printf("Error creating context for '%s' - %s\n", m.minfo.UID, err)
			return fmt.Errorf("Error creating context for '%s' - %s", m.minfo.UID, err)
		}
		/* following commemted line not work , no idea why */
		//m.inctx.SetOptions([]*gmf.Option{{"stream_loop", -1}})
		err = s.setupInputVideoDecodeCtx(m, VIDEO_CODEC_DECODER_NAME_X264)
		if err != nil {
			return fmt.Errorf("failed to setup input video decoder '%s'", m.minfo.UID)
		}

		err = s.setupInputAudioDecodeCtx(m,AUDIO_DECODE_CODEC_NAME_AAC)
		if err != nil {
			return fmt.Errorf("failed to setup input audio decoder '%s'", m.minfo.UID)
		}

		m.inctx.Dump()
		s.mstreams[m.minfo.UID] = m
		return err

	} else {
		return fmt.Errorf("stream already existed '%s'", m.minfo.UID)
	}

}

/* @vdecoderName decoder would have used same with input video stream , but in case of hardware decoding needed , so left this option*/
func (s *Streamer) setupInputVideoDecodeCtx(m *Mstream, vdecoderName string) error {

	var err error

	m.invstream, err = m.inctx.GetBestStream(gmf.AVMEDIA_TYPE_VIDEO)
	if err != nil {
		m.invstream.Free()
		log.Printf("No video stream found in '%s'\n", m.minfo.UID)
		return fmt.Errorf("No video stream found in '%s'", m.minfo.UID)
	}

	/* video stream extract and decode context set up */

	m.invCodec, err = gmf.FindDecoder(vdecoderName)
	if err != nil {
		log.Printf("coud not fine video stream decoder for '%s'\n", m.minfo.UID)
		return fmt.Errorf("coud not fine video stream decoder for '%s'", m.minfo.UID)
	}

	if m.invDecodecCtx = gmf.NewCodecCtx(m.invCodec); m.invDecodecCtx == nil {
		return fmt.Errorf("unable to create video codec context for %s", m.minfo.UID)
	}

	if err = m.invstream.GetCodecPar().ToContext(m.invDecodecCtx); err != nil {
		return fmt.Errorf("Failed to copy video decoder parameters to input decoder context  for %s", m.minfo.UID)
	}

	if err = m.invDecodecCtx.Open(nil); err != nil {
		return fmt.Errorf("Failed to open decoder for video stream  for %s", m.minfo.UID)
	}

	return err
}

func (s *Streamer) setupInputAudioDecodeCtx(m *Mstream ,adocederName string) error {

	var err error

	m.inastream, err = m.inctx.GetBestStream(gmf.AVMEDIA_TYPE_AUDIO)
	if err != nil {
		m.inastream.Free()
		log.Printf("No audio stream found in '%s'\n", m.minfo.UID)
		return fmt.Errorf("No audio stream found in '%s'", m.minfo.UID)
	}

	/* audio stream extract and decode context set up */
	m.inaCodec, err = gmf.FindDecoder(adocederName)
	if err != nil {
		log.Printf("coud not fine audio stream decoder for '%s'\n", m.minfo.UID)
		return fmt.Errorf("coud not fine audio stream decoder for '%s'", m.minfo.UID)
	}

	if m.inaDecodecCtx = gmf.NewCodecCtx(m.inaCodec); m.inaDecodecCtx == nil {
		return fmt.Errorf("unable to create audio codec context for %s", m.minfo.UID)
	}

	if err = m.inastream.GetCodecPar().ToContext(m.inaDecodecCtx); err != nil {
		return fmt.Errorf("Failed to copy audio decoder parameters to input decoder context  for %s", m.minfo.UID)
	}

	if err = m.inaDecodecCtx.Open(nil); err != nil {
		return fmt.Errorf("Failed to open decoder for audio stream  for %s", m.minfo.UID)
	}
	return err
}

func (s *Streamer) removeStream(suid string) {

	if s.streamExisted(suid) {

		/* execution order may matters */
		s.mstreams[suid].invDecodecCtx.Close()
		s.mstreams[suid].inaDecodecCtx.Close()
		s.mstreams[suid].invDecodecCtx.Free()
		s.mstreams[suid].inaDecodecCtx.Free()
		s.mstreams[suid].invCodec.Free()
		s.mstreams[suid].inaCodec.Free()
		s.mstreams[suid].invstream.Free()
		s.mstreams[suid].inastream.Free()
		s.mstreams[suid].inctx.Free() // input context Close called inside Free method
		delete(s.mstreams, suid)
	}
}

func (s *Streamer) streamExisted(suid string) bool {

	if _, ok := s.mstreams[suid]; ok {
		return true
	}
	return false
}

func (s *Streamer) isStreaming() bool {

	return len(s.currentStreamingUID) > 0
}

func (s *Streamer) initStreamingOutputCtx() error {

	var err error
	s.outctx, err = gmf.NewOutputCtx(STREAMER_PUSH_URL)
	if err != nil {
		s.outctx.Free()
		log.Printf("fail to create output context for streaming to server '%s' '%s' \n", STREAMER_PUSH_URL, err)
		return fmt.Errorf("fail to create output context for streaming to server '%s' '%s'", STREAMER_PUSH_URL, err)
	}
	return err
}

func (s *Streamer) setupOutputVideoEncodeCtx() error {

	var err error

	s.outvCodec, err = gmf.FindEncoder(VIDEO_CODEC_ENCODER_NAME_X264)
	if err != nil {
		return fmt.Errorf("output video encoder not found: '%s'", err)
	}

	if s.outvEncodeCtx = gmf.NewCodecCtx(s.outvCodec); s.outvEncodeCtx == nil {
		return fmt.Errorf("create output video encoder context failed: '%s'", err)
	}

	return err

}

func (s *Streamer) setupOutputVideoEncodeCtxOptions(suid string) error {

	var err error

	s.outvOptions = []gmf.Option{
		{Key: "time_base", Val: gmf.AVR{Num: 1, Den: STREAM_VIDEO_FRAMERATE}},
		{Key: "pixel_format", Val: VIDEO_OUTPUT_PIX_FORMAT},
		{Key: "video_size", Val: s.mstreams[suid].invDecodecCtx.GetVideoSize()},
		{Key: "b", Val: 500000},
	}
	s.outvEncodeCtx.SetPixFmt(VIDEO_OUTPUT_PIX_FORMAT)
	s.outvEncodeCtx.SetProfile(VIDEO_OUTPUT_264_PROFILE)
	s.outvEncodeCtx.SetOptions(s.outvOptions)

	if s.outctx.IsGlobalHeader() {
		s.outvEncodeCtx.SetFlag(gmf.CODEC_FLAG_GLOBAL_HEADER)
	}

	if s.outvCodec.IsExperimental() {
		s.outvEncodeCtx.SetStrictCompliance(gmf.FF_COMPLIANCE_EXPERIMENTAL)
	}

	if err = s.outvEncodeCtx.Open(nil); err != nil {
		return fmt.Errorf("Failed to open encoder for video stream")
	}

	return err

}

func (s *Streamer) setupOutputVideoStream() error {

	var err error

	if s.outvstream = s.outctx.NewStream(s.outvCodec); s.outvstream == nil {
		return fmt.Errorf("unable to create new video stream in output context: '%s'", err)
	}

	if s.outvstream.GetCodecPar().FromContext(s.outvEncodeCtx); err != nil {

		return fmt.Errorf("Failed to copy output video encoder parameters to output video stream - %s", err)
	}

	s.outvstream.SetTimeBase(gmf.AVR{Num: 1, Den: STREAM_VIDEO_FRAMERATE})
	s.outvstream.SetRFrameRate(gmf.AVR{Num: STREAM_VIDEO_FRAMERATE, Den: 1})

	return err
}

func (s *Streamer) setupOutputAudioEncodeCtx() error {

	var err error

	s.outaCodec, err = gmf.FindEncoder(AUDIO_ENCODE_CODEC_NAME_AAC)
	if err != nil {
		return fmt.Errorf("audio encoder not found: '%s'", err)
	}

	if s.outaEncodeCtx = gmf.NewCodecCtx(s.outaCodec); s.outaEncodeCtx == nil {
		return fmt.Errorf("create audio encoder context failed: '%s'", err)
	}
	return err
}

func (s *Streamer) setupOutputAudioEncodeCtxOptions() error {

	var err error

	s.outaOptions = []gmf.Option{
		{Key: "time_base", Val: gmf.AVR{Num: 1, Den: STREAM_VIDEO_FRAMERATE}},
		//{Key: "ar", Val: AUDIO_AAC_OUTPUT_SAMPLE_RATE},
		{Key: "ac", Val: AUDIO_AAC_OUTPUT_CHANNELS},
		{Key: "channel_layout", Val: s.outaEncodeCtx.GetDefaultChannelLayout(AUDIO_AAC_OUTPUT_CHANNELS)},
	}
	s.outaEncodeCtx.SetSampleFmt(AUDIO_AAC_OUTPUT_SAMPLE_FORMAT)
	s.outaEncodeCtx.SelectSampleRate()
	s.outaEncodeCtx.SetOptions(s.outaOptions)

	if s.outctx.IsGlobalHeader() {
		s.outaEncodeCtx.SetFlag(gmf.CODEC_FLAG_GLOBAL_HEADER)
	}

	if s.outaCodec.IsExperimental() {
		s.outaEncodeCtx.SetStrictCompliance(gmf.FF_COMPLIANCE_EXPERIMENTAL)
	}

	if err = s.outaEncodeCtx.Open(nil); err != nil {
		return fmt.Errorf("Failed to open encoder for audio stream  for")
	}

	return err
}

func (s *Streamer) setupOutputAudioStream() error {

	var err error

	if s.outastream = s.outctx.NewStream(s.outaCodec); s.outastream == nil {
		return fmt.Errorf("unable to create new audio stream in output context: '%s'", err)
	}

	if s.outastream.GetCodecPar().FromContext(s.outaEncodeCtx); err != nil {

		return fmt.Errorf("Failed to copy audio encoder parameters to output stream - %s", err)
	}
	s.outastream.SetTimeBase(gmf.AVR{Num: 1, Den: STREAM_VIDEO_FRAMERATE})
	return err
}

func (s *Streamer) startStreaming(mInfo StreamInfo) error {

	var err error
	suid := mInfo.UID
	if s.currentStreamingUID != suid {

		err = s.initStreamingOutputCtx()
		if err != nil {
			return err
		}
		err = s.setupOutputVideoEncodeCtx()
		if err != nil {
			return err
		}

		err = s.setupOutputVideoEncodeCtxOptions(suid)
		if err != nil {
			return err
		}

		err = s.setupOutputVideoStream()
		if err != nil {
			return err
		}

		err =s.initWaterMark(suid)
		if err != nil {
			return err
		}	

		err = s.setupOutputAudioEncodeCtx()
		if err != nil {
			return err
		}

		err = s.setupOutputAudioEncodeCtxOptions()
		if err != nil {
			return err
		}

		err = s.setupOutputAudioStream()
		if err != nil {
			return err
		}

		s.currentStreamingUID = suid

		err =s.initMp3s()
		if err != nil {
			return err
		}	

		if err := s.outctx.WriteHeader(); err != nil {
			return fmt.Errorf("error writing header - %s", err)
		}

		var (
			pkt       *gmf.Packet
			streamIdx int
			frames []*gmf.Frame
			frame *gmf.Frame
			errInt int		
			filterInitedOnce bool =false	
		)

		for{		

			pkt, err = s.mstreams[suid].inctx.GetNextPacket()
			if err != nil && err != io.EOF {
				if pkt != nil {
					pkt.Free()
				}
				return fmt.Errorf("error getting next packet - %s", err)
			} else if err != nil && pkt == nil {
				fmt.Printf("=== flushing \n")
				break
			}
			
			streamIdx = pkt.StreamIndex()


			if streamIdx == s.mstreams[suid].inastream.Index() {


				if len(s.mp3InCtxs) != 0{

					if frames,err=s.getMp3sFrames();err != nil{

						
					}

				}else{
	
					frame, errInt = s.mstreams[suid].inaDecodecCtx.Decode2(pkt)

					if errInt < 0 && gmf.AvErrno(errInt) == syscall.EAGAIN {
						continue
					} else if errInt == gmf.AVERROR_EOF {
						return fmt.Errorf("EOF in audio Decode2, handle it\n")
					} else if errInt < 0 {
						return fmt.Errorf("Unexpected error audio - %s\n", gmf.AvError(errInt))
					}

					/*resampleOptions := []*gmf.Option{
						{Key: "in_channel_layout", Val: s.mstreams[suid].inaDecodecCtx.ChannelLayout()},
						{Key: "out_channel_layout", Val: s.outaEncodeCtx.ChannelLayout()},
						{Key: "in_sample_rate", Val: s.mstreams[suid].inaDecodecCtx.SampleRate()},
						{Key: "out_sample_rate", Val: s.outaEncodeCtx.SampleRate()},
						{Key: "in_sample_fmt", Val: gmf.SampleFormat(s.mstreams[suid].inaDecodecCtx.SampleFmt())},
						{Key: "out_sample_fmt", Val: gmf.SampleFormat(s.outaEncodeCtx.SampleFmt())},
					}
			
					if s.outastream.SwrCtx, err = gmf.NewSwrCtx(resampleOptions, s.outaEncodeCtx.Channels(), s.outaEncodeCtx.SampleFmt()); err != nil {
						panic(err)
					}
					s.outastream.AvFifo = gmf.NewAVAudioFifo(s.mstreams[suid].inaDecodecCtx.SampleFmt(), s.mstreams[suid].inaDecodecCtx.Channels(), 1024)

					frames = gmf.DefaultResampler(s.outastream, []*gmf.Frame{frame}, false)
	*/
				}
				

				packets, err := s.outaEncodeCtx.Encode([]*gmf.Frame{frame}, -1)

				for _, op := range packets {

					if op.Dts() != gmf.AV_NOPTS_VALUE || op.Pts() != gmf.AV_NOPTS_VALUE {

						gmf.RescaleTs(op, s.mstreams[suid].inastream.TimeBase(), s.outastream.TimeBase())
						op.SetStreamIndex(s.outastream.Index())
						if err = s.outctx.WritePacket(op); err != nil {
							break
						}
					}
					op.Free()
				}		

			}

			if streamIdx == s.mstreams[suid].invstream.Index() {

				frame, errInt = s.mstreams[suid].invDecodecCtx.Decode2(pkt)

				if errInt < 0 && gmf.AvErrno(errInt) == syscall.EAGAIN {
					continue
				} else if errInt == gmf.AVERROR_EOF {
					return fmt.Errorf("EOF in video Decode2, handle it\n")
				} else if errInt < 0 {
					return fmt.Errorf("Unexpected error video - %s\n", gmf.AvError(errInt))
				}


				if s.watermarkFilter != nil{

					if !filterInitedOnce{

						if err := s.watermarkFilter.AddFrame(frame, 0, 0); err != nil {
							return fmt.Errorf("%s\n", err)
						}
						filterInitedOnce=true
	
						if err := s.watermarkFilter.AddFrame(s.waterMarkImgFrame, 1, 4); err != nil {
							return fmt.Errorf("%s\n", err)
						}
						s.watermarkFilter.RequestOldest()
						s.watermarkFilter.Close(1)
	
					}else{
	
						if err := s.watermarkFilter.AddFrame(frame, 0, 4); err != nil {
							return fmt.Errorf("%s\n", err)
	
						}
					}
	
					if frames, err = s.watermarkFilter.GetFrame(); err != nil && len(frames) == 0 {
						fmt.Printf("GetFrame() returned '%s', continue\n", err)
					}
	
					
				}

				packets, err := s.outvEncodeCtx.Encode(frames, -1)

				for _, op := range packets {

					if op.Dts() != gmf.AV_NOPTS_VALUE || op.Pts() != gmf.AV_NOPTS_VALUE {

						//op.SetDuration(int64(s.outvstream.TimeBase().AVR().Den / s.outvstream.TimeBase().AVR().Num / s.mstreams[suid].invstream.GetAvgFrameRate().AVR().Num * s.mstreams[suid].invstream.GetAvgFrameRate().AVR().Den))
						gmf.RescaleTs(op, s.mstreams[suid].invstream.TimeBase(), s.outvstream.TimeBase())
						op.SetStreamIndex(s.outvstream.Index())
						if err = s.outctx.WritePacket(op); err != nil {
							break
						}

					}
					op.Free()
				}

			}

			if frame != nil {
				frame.Free()
			}

			for _, frame := range frames {
				if frame != nil {
					frame.Free()
				}
			}

			if pkt != nil {
				pkt.Free()
			}

		}

		s.outctx.WriteTrailer()

		s.outctx.Dump()

	} else {
		//return fmt.Printf("already streaming this '%s'\n", suid)
	}
	return err
}

func (s *Streamer) stopStreaming() {

	if s.watermarkFilter != nil{
		s.clearUpWaterMarkRes()
	}

	if len(s.mp3InCtxs) > 0{

		s.clearUpMp3sRes()
	}
	s.outvEncodeCtx.Close()
	s.outaEncodeCtx.Close()
	s.outvOptions = nil
	s.outaOptions = nil
	s.outvEncodeCtx.Free()
	s.outaEncodeCtx.Free()
	s.outvCodec.Free()
	s.outaCodec.Free()
	s.outvstream.Free()
	s.outastream.Free()
	s.outctx.Free()
	s.currentStreamingUID = ""
}


func main() {
	Streamer := &Streamer{mstreams: make(map[string]*Mstream)}
	Minfo := StreamInfo{
		Schema:   "",
		Vhost:    "",
		AppName:  "live",
		StreamId: "text",
		UID:      "/Users/s1ngular/GoWork/src/github.com/organicio/bbb.mp4",
	}
	var err error
	err = Streamer.addStream(Minfo)
	if err != nil {
		fmt.Println(err)
	}

	err = Streamer.startStreaming(Minfo)
	fmt.Println(err)
	Streamer.stopStreaming()
}
