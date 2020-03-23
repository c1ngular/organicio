package mainBackup

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/3d0c/gmf"
)

const (
	VIDEO_ENCODE_CODEC_NAME_X264 = "libx264"
	VIDEO_DECODE_CODEC_NAME_X264 = "h264"

	AUDIO_DECODE_CODEC_NAME_AAC = "aac"
	AUDIO_ENCODE_CODEC_NAME_AAC = "aac"
	AUDIO_DECODE_CODEC_NAME_MP3 = "mp3"

	STREAM_RESOLUTION_HIGHT  = 21
	STREAM_RESOLUTION_MEDIUM = 28
	STREAM_RESOLUTION_LOW    = 34

	STREAM_MAX_BITRATE     = "1M"
	STREAM_BUFSIZE         = "2M"
	STREAM_AUDIO_BITRATE   = "64K"
	STREAM_VIDEO_BITRATE   = "900K"
	STREAM_VIDEO_FRAMERATE = 24

	STREAMER_UUID     = "sdfrweqrewqr0943rasdfds"
	STREAMER_SECRET   = "FDQER345435435"
	STREAMER_PUSH_URL = "/Users/s1ngular/GoWork/src/github.com/organicio/tesdtt.mp4"

	WATERMARK_IMG_URL               = "/Users/s1ngular/GoWork/src/github.com/organicio/watermark.png"
	WATERMARK_POSITION_BOTTOM_RIGHT = "overlay=main_w-overlay_w-10:main_h-overlay_h-10"
	WATERMARK_POSITION_BOTTOM_LEFT  = "overlay=10:main_h-overlay_h-10"
	WATERMARK_POSITION_TOP_RIGHT    = "overlay=main_w-overlay_w-10:10"
	WATERMARK_POSITION_TOP_LEFT     = "overlay=10:10"
	MP3S_FOLDER_PATH                = "/Users/s1ngular/GoWork/src/github.com/organicio/mp3s"
)

var (
	AUDIO_AAC_OUTPUT_SAMPLE_FORMAT int32 = gmf.AV_SAMPLE_FMT_FLTP
	VIDEO_OUTPUT_PIX_FORMAT        int32 = gmf.AV_PIX_FMT_YUV420P
	VIDEO_OUTPUT_264_PROFILE       int   = gmf.FF_PROFILE_H264_BASELINE
)

type StreamInfo struct {
	Schema   string `json:"schema"`
	Vhost    string `json:"vhost"`
	AppName  string `json:"app"`
	StreamId string `json:"stream"`
	UID      string
}

type Mstream struct {
	minfo     StreamInfo
	inctx     *gmf.FmtCtx
	inastream *gmf.Stream
	invstream *gmf.Stream
}

type Streamer struct {
	mstreams            map[string]*Mstream
	currentStreamingUID string

	outctx        *gmf.FmtCtx
	outvstream    *gmf.Stream
	outastream    *gmf.Stream
	outaEncodeCtx *gmf.CodecCtx
	outvEncodeCtx *gmf.CodecCtx

	waterMarkImageCtx      *gmf.FmtCtx
	waterMarkImageStream   *gmf.Stream
	waterMarkOverlayFilter *gmf.Filter
	waterMarkImagePacket   *gmf.Packet
	waterMarkImageFrame    *gmf.Frame

	mp3InCtxs                []*gmf.FmtCtx
	mp3InStreams             []*gmf.Stream
	currentStreamingMp3Index int

	mux sync.Mutex
}

func (s *Streamer) initMp3s() error {

	var err error
	var inctx *gmf.FmtCtx
	var mp3Stream *gmf.Stream
	var mp3Decodec *gmf.Codec
	var mp3DecodecCtx *gmf.CodecCtx

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
			fmt.Printf("Error creating mp3 input context for %s - %s\n", mp3.Name(), err)
		} else {
			s.mp3InCtxs = append(s.mp3InCtxs, inctx)
		}

		mp3Stream, err = inctx.GetBestStream(gmf.AVMEDIA_TYPE_AUDIO)
		if err != nil {
			mp3Stream.Free()
			log.Printf("No audio stream found in mp3 : '%s'\n", mp3.Name())
			return fmt.Errorf("No audio stream found in mp3 : '%s'", mp3.Name())
		} else {

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
				return fmt.Errorf("Failed to copy mp3 decoder parameters to input decoder context  for %s", mp3.Name())
			}

			if err = mp3DecodecCtx.Open(nil); err != nil {
				return fmt.Errorf("Failed to open decoder for mp3 stream  for %s", mp3.Name())
			}

			s.mp3InStreams = append(s.mp3InStreams, mp3Stream)
		}

	}

	return err
}

func (s *Streamer) getMp3sFrames() ([]*gmf.Frame, error) {

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
			fmt.Print("create NEw SwR")
			panic(err)
		}
		s.outastream.AvFifo = gmf.NewAVAudioFifo(s.mp3InStreams[s.currentStreamingMp3Index].CodecCtx().SampleFmt(), s.mp3InStreams[s.currentStreamingMp3Index].CodecCtx().Channels(), 1024)

	}

	var actx = s.mp3InCtxs[s.currentStreamingMp3Index]

	pkt, err = actx.GetNextPacket()
	if err != nil && err != io.EOF {
		if pkt != nil {
			pkt.Free()
		}
		return frames, fmt.Errorf("error getting next packet - %s", err)

	} else if err != nil && pkt == nil {

		actx.SeekFile(s.mp3InStreams[s.currentStreamingMp3Index], 0, int64(actx.Duration()), 0)
		fmt.Printf(" reaching end of current mp3 stream \n")
		if s.currentStreamingMp3Index == len(s.mp3InCtxs)-1 {
			s.currentStreamingMp3Index = 0
		} else {
			s.currentStreamingMp3Index++
		}

		return s.getMp3sFrames()

	}

	streamIdx := pkt.StreamIndex()
	if streamIdx == s.mp3InStreams[s.currentStreamingMp3Index].Index() {

		frame, errInt = s.mp3InStreams[s.currentStreamingMp3Index].CodecCtx().Decode2(pkt)

		if errInt < 0 && gmf.AvErrno(errInt) == syscall.EAGAIN {
			return s.getMp3sFrames()
		} else if errInt == gmf.AVERROR_EOF {
			return frames, fmt.Errorf("EOF in mp3 audio Decode2, handle it\n")
		} else if errInt < 0 {
			return frames, fmt.Errorf("Unexpected error mp3 audio - %s\n", gmf.AvError(errInt))
		}

		frames = gmf.DefaultResampler(s.outastream, []*gmf.Frame{frame}, false)
		return frames, nil

	} else {
		return s.getMp3sFrames()
	}

}

func (s *Streamer) releaseMp3sResource() {

	for i, _ := range s.mp3InCtxs {

		s.mp3InStreams[i].Free()
		s.mp3InCtxs[i].Free()
	}
}

func (s *Streamer) initWaterMarkWithInputVideoStream(filename string, invstream *gmf.Stream, position string) error {

	var err error
	s.waterMarkImageCtx, err = gmf.NewInputCtx(filename)
	if err != nil {
		s.waterMarkImageCtx.Free()
		return fmt.Errorf("failed to create watermark input context for %s", filename)
	}

	s.waterMarkImageStream, err = s.waterMarkImageCtx.GetBestStream(gmf.AVMEDIA_TYPE_VIDEO)
	if err != nil {
		s.waterMarkImageStream.Free()
		return fmt.Errorf("failed to create watermark input stream for %s", filename)
	}

	s.waterMarkOverlayFilter, err = gmf.NewFilter(position, []*gmf.Stream{invstream, s.waterMarkImageStream}, s.outvstream, []*gmf.Option{})
	if err != nil {
		s.waterMarkOverlayFilter.Release()
		return err
	}

	err = s.decodeWaterMarkImageFrame()
	return err
}

/* seems static picture packet number  and frame number is alway 1 ? to be figured out */

func (s *Streamer) decodeWaterMarkImageFrame() error {
	var err error
	var pkt *gmf.Packet

	pkt, err = s.waterMarkImageCtx.GetNextPacket()
	if err != nil && err != io.EOF {
		if pkt != nil {
			pkt.Free()
		}
		return err
	} else if err != nil && pkt == nil {
		return err
	}

	s.waterMarkImagePacket = pkt
	s.waterMarkImageFrame, _ = s.waterMarkImageStream.CodecCtx().Decode2(s.waterMarkImagePacket)

	return err
}

func (s *Streamer) releaseWaterMarkResource() {

	if s.waterMarkImageFrame != nil {
		s.waterMarkImageFrame.Free()
	}
	if s.waterMarkImagePacket != nil {
		s.waterMarkImagePacket.Free()
	}
	s.waterMarkOverlayFilter.Release()
	s.waterMarkImageStream.Free()
	s.waterMarkImageCtx.Free()
}

func (s *Streamer) addNewStream(mInfo StreamInfo) error {

	var err error

	m := &Mstream{minfo: mInfo}

	if !s.streamExisted(m.minfo.UID) {
		m.inctx, err = gmf.NewInputCtx(m.minfo.UID)
		if err != nil {
			m.inctx.Free()
			fmt.Printf("Error creating context for '%s' - %s\n", m.minfo.UID, err)
			return fmt.Errorf("Error creating context for '%s' - %s", m.minfo.UID, err)
		}

		err = s.setupInputVideoDecodeCtx(m, VIDEO_DECODE_CODEC_NAME_X264)
		if err != nil {
			return fmt.Errorf("failed to setup input video decoder '%s'", m.minfo.UID)
		}

		err = s.setupInputAudioDecodeCtx(m, AUDIO_DECODE_CODEC_NAME_AAC)
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

func (s *Streamer) setupInputVideoDecodeCtx(m *Mstream, vdecoderName string) error {

	var err error
	var invCodec *gmf.Codec
	var invDecodeCtx *gmf.CodecCtx

	m.invstream, err = m.inctx.GetBestStream(gmf.AVMEDIA_TYPE_VIDEO)
	if err != nil {
		m.invstream.Free()
		log.Printf("No video stream found in '%s'\n", m.minfo.UID)
		return fmt.Errorf("No video stream found in '%s'", m.minfo.UID)
	}

	/* video stream extract and decode context set up */

	invCodec, err = gmf.FindDecoder(vdecoderName)
	if err != nil {
		log.Printf("coud not fine video stream decoder for '%s'\n", m.minfo.UID)
		return fmt.Errorf("coud not fine video stream decoder for '%s'", m.minfo.UID)
	}

	if invDecodeCtx = gmf.NewCodecCtx(invCodec); invDecodeCtx == nil {
		return fmt.Errorf("unable to create video codec context for %s", m.minfo.UID)
	}

	if err = m.invstream.GetCodecPar().ToContext(invDecodeCtx); err != nil {
		return fmt.Errorf("Failed to copy video decoder parameters to input decoder context  for %s", m.minfo.UID)
	}

	if err = invDecodeCtx.Open(nil); err != nil {
		return fmt.Errorf("Failed to open decoder for video stream  for %s", m.minfo.UID)
	}

	return err
}

func (s *Streamer) setupInputAudioDecodeCtx(m *Mstream, adecoderName string) error {

	var err error
	var inaCodec *gmf.Codec
	var inaDecodeCtx *gmf.CodecCtx

	m.inastream, err = m.inctx.GetBestStream(gmf.AVMEDIA_TYPE_AUDIO)
	if err != nil {
		m.inastream.Free()
		log.Printf("No audio stream found in '%s'\n", m.minfo.UID)
		return fmt.Errorf("No audio stream found in '%s'", m.minfo.UID)
	}

	/* audio stream extract and decode context set up */
	inaCodec, err = gmf.FindDecoder(adecoderName)
	if err != nil {
		log.Printf("coud not fine audio stream decoder for '%s'\n", m.minfo.UID)
		return fmt.Errorf("coud not fine video stream decoder for '%s'", m.minfo.UID)
	}

	if inaDecodeCtx = gmf.NewCodecCtx(inaCodec); inaDecodeCtx == nil {
		return fmt.Errorf("unable to create audio codec context for %s", m.minfo.UID)
	}

	if err = m.inastream.GetCodecPar().ToContext(inaDecodeCtx); err != nil {
		return fmt.Errorf("Failed to copy audio decoder parameters to input decoder context  for %s", m.minfo.UID)
	}

	if err = inaDecodeCtx.Open(nil); err != nil {
		return fmt.Errorf("Failed to open decoder for audio stream  for %s", m.minfo.UID)
	}
	return err
}

func (s *Streamer) removeStream(suid string) {

	if s.streamExisted(suid) {

		/* execution order may matters */
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

func (s *Streamer) initStreamingOutputCtx(outputUrl string) error {

	var err error
	s.outctx, err = gmf.NewOutputCtx(outputUrl)
	if err != nil {
		s.outctx.Free()
		log.Printf("fail to create output context for streaming to server '%s' '%s' \n", STREAMER_PUSH_URL, err)
		return fmt.Errorf("fail to create output context for streaming to server '%s' '%s'", STREAMER_PUSH_URL, err)
	}
	return err
}

func (s *Streamer) setupOutputVideoEncodeCtxWithOptions(vencoderName string, invstream *gmf.Stream) (*gmf.CodecCtx, error) {

	var err error
	var outvCodec *gmf.Codec
	var outvEncodeCtx *gmf.CodecCtx
	outvCodec, err = gmf.FindEncoder(vencoderName)
	if err != nil {
		return outvEncodeCtx, fmt.Errorf("output video encoder not found: '%s'", err)
	}

	if outvEncodeCtx = gmf.NewCodecCtx(outvCodec); outvEncodeCtx == nil {
		return outvEncodeCtx, fmt.Errorf("create output video encoder context failed: '%s'", err)
	}

	outvEncodeCtx.SetTimeBase(gmf.AVR{Num: 1, Den: STREAM_VIDEO_FRAMERATE})
	outvEncodeCtx.SetPixFmt(VIDEO_OUTPUT_PIX_FORMAT)
	outvEncodeCtx.SetDimension(invstream.CodecCtx().Width(), invstream.CodecCtx().Height())
	outvEncodeCtx.SetProfile(VIDEO_OUTPUT_264_PROFILE)

	if s.outctx.IsGlobalHeader() {
		outvEncodeCtx.SetFlag(gmf.CODEC_FLAG_GLOBAL_HEADER)
	}

	if outvCodec.IsExperimental() {
		outvEncodeCtx.SetStrictCompliance(gmf.FF_COMPLIANCE_EXPERIMENTAL)
	}

	if err = outvEncodeCtx.Open(nil); err != nil {
		return outvEncodeCtx, fmt.Errorf("Failed to open encoder for video stream")
	}

	return outvEncodeCtx, err

}

func (s *Streamer) createOutputVideoStreamWithEncodeCtx(vencodeCtx *gmf.CodecCtx) error {

	var err error

	if s.outvstream = s.outctx.NewStream(vencodeCtx.Codec()); s.outvstream == nil {
		return fmt.Errorf("unable to create new video stream in output context: '%s'", err)
	}

	if s.outvstream.GetCodecPar().FromContext(vencodeCtx); err != nil {

		return fmt.Errorf("Failed to copy output video encoder parameters to output video stream - %s", err)
	}

	s.outvstream.SetTimeBase(gmf.AVR{Num: 1, Den: STREAM_VIDEO_FRAMERATE})
	s.outvstream.SetRFrameRate(gmf.AVR{Num: STREAM_VIDEO_FRAMERATE, Den: 1})

	return err
}

func (s *Streamer) setupOutputAudioEncodeCtxWithOptions(aencoderName string, inastream *gmf.Stream) (*gmf.CodecCtx, error) {

	var err error
	var outaCodec *gmf.Codec
	var outaEncodeCtx *gmf.CodecCtx

	outaCodec, err = gmf.FindEncoder(aencoderName)
	if err != nil {
		return outaEncodeCtx, fmt.Errorf("audio encoder not found: '%s'", err)
	}

	if outaEncodeCtx = gmf.NewCodecCtx(outaCodec); outaEncodeCtx == nil {
		return outaEncodeCtx, fmt.Errorf("create audio encoder context failed: '%s'", err)
	}

	outaEncodeCtx.SetTimeBase(gmf.AVR{Num: 1, Den: 22050})
	outaEncodeCtx.SetSampleRate(22050)
	outaEncodeCtx.SetChannels(2)
	outaEncodeCtx.SetChannelLayout(outaEncodeCtx.GetDefaultChannelLayout(2))
	outaEncodeCtx.SetSampleFmt(AUDIO_AAC_OUTPUT_SAMPLE_FORMAT)

	if s.outctx.IsGlobalHeader() {
		outaEncodeCtx.SetFlag(gmf.CODEC_FLAG_GLOBAL_HEADER)
	}

	if outaCodec.IsExperimental() {
		outaEncodeCtx.SetStrictCompliance(gmf.FF_COMPLIANCE_EXPERIMENTAL)
	}

	if err = outaEncodeCtx.Open(nil); err != nil {
		return outaEncodeCtx, fmt.Errorf("Failed to open encoder for audio stream")
	}

	return outaEncodeCtx, err
}

func (s *Streamer) createOutputAudioStreamWithEncodeCtx(aencodeCtx *gmf.CodecCtx) error {

	var err error

	if s.outastream = s.outctx.NewStream(aencodeCtx.Codec()); s.outastream == nil {
		return fmt.Errorf("unable to create new audio stream in output context: '%s'", err)
	}

	if s.outastream.GetCodecPar().FromContext(aencodeCtx); err != nil {

		return fmt.Errorf("Failed to copy audio encoder parameters to output stream - %s", err)
	}
	s.outastream.SetTimeBase(gmf.AVR{Num: 1, Den: 22050})
	return err
}

func (s *Streamer) startStreaming(mInfo StreamInfo) error {

	var err error
	suid := mInfo.UID
	if s.currentStreamingUID != suid {

		err = s.initStreamingOutputCtx(STREAMER_PUSH_URL)
		if err != nil {
			return err
		}
		s.outvEncodeCtx, err = s.setupOutputVideoEncodeCtxWithOptions(VIDEO_ENCODE_CODEC_NAME_X264, s.mstreams[suid].invstream)
		if err != nil {
			return err
		}

		err = s.createOutputVideoStreamWithEncodeCtx(s.outvEncodeCtx)
		if err != nil {
			return err
		}

		if WATERMARK_IMG_URL != "" {

			err = s.initWaterMarkWithInputVideoStream(WATERMARK_IMG_URL, s.mstreams[suid].invstream, WATERMARK_POSITION_TOP_RIGHT)
			if err != nil {
				return err
			}
		}

		s.outaEncodeCtx, err = s.setupOutputAudioEncodeCtxWithOptions(AUDIO_ENCODE_CODEC_NAME_AAC, s.mstreams[suid].inastream)
		if err != nil {
			return err
		}

		err = s.createOutputAudioStreamWithEncodeCtx(s.outaEncodeCtx)
		if err != nil {
			return err
		}

		if MP3S_FOLDER_PATH != "" {

			err = s.initMp3s()
			if err != nil {
				return err
			}
		}

		s.currentStreamingUID = suid
		//s.outctx.SetStartTime(0)
		if err := s.outctx.WriteHeader(); err != nil {
			return fmt.Errorf("error writing header - %s", err)
		}

		var (
			pkt              *gmf.Packet
			streamIdx        int
			frames           []*gmf.Frame
			frame            *gmf.Frame
			errInt           int
			filterInitedOnce bool = false
		)

		for i := 0; i < 200; i++ {

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

				if len(s.mp3InStreams) != 0 {

					frames, err = s.getMp3sFrames()
					if err != nil {
						return err
					}
					fmt.Printf(" \n mp3 resample frame length: %d \n", len(frames))

				} else {

					if s.outastream.SwrCtx == nil {

						outgoingAudioResampleOptions := []*gmf.Option{
							{Key: "in_channel_layout", Val: s.mstreams[suid].inastream.CodecCtx().ChannelLayout()},
							{Key: "out_channel_layout", Val: s.outaEncodeCtx.ChannelLayout()},
							{Key: "in_sample_rate", Val: s.mstreams[suid].inastream.CodecCtx().SampleRate()},
							{Key: "out_sample_rate", Val: s.outaEncodeCtx.SampleRate()},
							{Key: "in_sample_fmt", Val: gmf.SampleFormat(s.mstreams[suid].inastream.CodecCtx().SampleFmt())},
							{Key: "out_sample_fmt", Val: gmf.SampleFormat(s.outaEncodeCtx.SampleFmt())},
						}

						if s.outastream.SwrCtx, err = gmf.NewSwrCtx(outgoingAudioResampleOptions, s.outaEncodeCtx.Channels(), s.outaEncodeCtx.SampleFmt()); err != nil {
							fmt.Print("create NEw SwR")
							panic(err)
						}
						s.outastream.AvFifo = gmf.NewAVAudioFifo(s.mstreams[suid].inastream.CodecCtx().SampleFmt(), s.mstreams[suid].inastream.CodecCtx().Channels(), 1024)

					}

					if frames, err = s.mstreams[suid].inastream.CodecCtx().Decode(pkt); err != nil {
						return fmt.Errorf(" deocding audio failed : %s", err)
					}
				}
				frames = gmf.DefaultResampler(s.outastream, frames, true)
				packets, err := s.outaEncodeCtx.Encode(frames, -1)
				if err != nil {
					return fmt.Errorf("\n audio encode failed %s\n", err)
				}

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

				frame, errInt = s.mstreams[suid].invstream.CodecCtx().Decode2(pkt)

				if errInt < 0 && gmf.AvErrno(errInt) == syscall.EAGAIN {
					continue
				} else if errInt == gmf.AVERROR_EOF {
					return fmt.Errorf("EOF in video Decode2, handle it\n")
				} else if errInt < 0 {
					return fmt.Errorf("Unexpected error video - %s\n", gmf.AvError(errInt))
				}

				if s.waterMarkOverlayFilter != nil {

					if !filterInitedOnce {

						if err := s.waterMarkOverlayFilter.AddFrame(frame, 0, 0); err != nil {
							return fmt.Errorf("%s\n", err)
						}
						filterInitedOnce = true

						if err := s.waterMarkOverlayFilter.AddFrame(s.waterMarkImageFrame, 1, 4); err != nil {
							return fmt.Errorf("%s\n", err)
						}
						s.waterMarkOverlayFilter.RequestOldest()
						s.waterMarkOverlayFilter.Close(1)

					} else {

						if err := s.waterMarkOverlayFilter.AddFrame(frame, 0, 4); err != nil {
							return fmt.Errorf("%s\n", err)

						}
					}

					if frames, err = s.waterMarkOverlayFilter.GetFrame(); err != nil && len(frames) == 0 {
						fmt.Printf("GetFrame() returned '%s', continue\n", err)
					}

				} else {
					frames = append(frames, frame)
				}

				packets, err := s.outvEncodeCtx.Encode(frames, -1)
				if err != nil {
					return err
				}

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

	if s.waterMarkOverlayFilter != nil {
		s.releaseWaterMarkResource()
	}

	if len(s.mp3InCtxs) != 0 {

		s.releaseMp3sResource()
	}

	s.outvEncodeCtx.Close()
	s.outaEncodeCtx.Close()
	s.outvEncodeCtx.Free()
	s.outaEncodeCtx.Free()
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
		UID:      "rtmp://202.69.69.180:443/webcast/bshdlive-pc",
	}
	var err error
	err = Streamer.addNewStream(Minfo)
	if err != nil {
		fmt.Println(err)
	}

	err = Streamer.startStreaming(Minfo)
	fmt.Println(err)
	Streamer.stopStreaming()
}