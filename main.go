package main

import (
	"fmt"
	"io"
	"log"
	"sync"
	"syscall"
	"github.com/3d0c/gmf"
)

const (

	VIDEO_ENCODE_CODEC_NAME_X264    = "libx264"
	VIDEO_DECODE_CODEC_NAME_X264    = "h264"

	AUDIO_DECODE_CODEC_NAME_AAC = "libfdk_aac"
	AUDIO_ENCODE_CODEC_NAME_AAC = "libfdk_aac"
	AUDIO_DECODE_CODEC_NAME_MP3 = "mp3"
)

var (
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

	WATERMARK_IMG_URL="/Users/s1ngular/GoWork/src/github.com/organicio/watermark.png"
	WATERMARK_POSITION_BOTTOM_RIGHT="overlay=main_w-overlay_w-10:main_h-overlay_h-10"
	WATERMARK_POSITION_BOTTOM_LEFT="overlay=10:main_h-overlay_h-10"
	WATERMARK_POSITION_TOP_RIGHT="overlay=main_w-overlay_w-10:10"
	WATERMARK_POSITION_TOP_LEFT="overlay=10:10"
	MP3S_FOLDER_PATH="/Users/s1ngular/GoWork/src/github.com/organicio/mp3s"
)

var (

	AUDIO_AAC_OUTPUT_SAMPLE_FORMAT int32 =gmf.AV_SAMPLE_FMT_FLTP
	VIDEO_OUTPUT_PIX_FORMAT int32 =gmf.AV_PIX_FMT_YUV420P
	VIDEO_OUTPUT_264_PROFILE int =gmf.FF_PROFILE_H264_BASELINE
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
	inaDecodeCtx *gmf.CodecCtx
	invDecodeCtx *gmf.CodecCtx
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

	waterMarkImageCtx *gmf.FmtCtx
	wwaterMarkImageStream *gmf.Stream
	waterMarkOverlayFilter *gmf.Filter
	waterMarkImagePacket *gmf.Packet
	waterMarkImageFrame *gmf.Frame

	mux           sync.Mutex
}

func (s *Streamer) initWaterMarkWithInputVideoStream(filename string,suid string , position string)error{

	var err error
	s.waterMarkImageCtx, err = gmf.NewInputCtx(filename)
	if(err != nil){
		s.waterMarkImageCtx.Free()
		return fmt.Errorf("failed to create watermark input context for %s",filename)
	}

	s.wwaterMarkImageStream,err=s.waterMarkImageCtx.GetBestStream(gmf.AVMEDIA_TYPE_VIDEO)
	if(err != nil){
		s.wwaterMarkImageStream.Free()
		return fmt.Errorf("failed to create watermark input stream for %s",filename)
	}


	s.waterMarkOverlayFilter, err = gmf.NewFilter(position, []*gmf.Stream{s.mstreams[suid].invstream, s.wwaterMarkImageStream}, s.outvstream, []*gmf.Option{})
	if err != nil {
		s.waterMarkOverlayFilter.Release()
		return err
	}	

	err=s.decodeWaterMarkImageFrame()
	return err
}

/* seems static picture packet number  and frame number is alway 1 ? to be figured out */

func (s *Streamer) decodeWaterMarkImageFrame() error {
	var err error
	var pkt *gmf.Packet

	
	pkt, err =s.waterMarkImageCtx.GetNextPacket()
	if err != nil && err != io.EOF {
		if pkt != nil {
			pkt.Free()
		}
		return err
	} else if err != nil && pkt == nil {
		return err
	}

	s.waterMarkImagePacket=pkt
	s.waterMarkImageFrame, _=s.wwaterMarkImageStream.CodecCtx().Decode2(s.waterMarkImagePacket)
	
	return err
}

func (s *Streamer) releaseWaterMarkResource(){

	if s.waterMarkImageFrame != nil{
		s.waterMarkImageFrame.Free()
	}
	if s.waterMarkImagePacket != nil{
		s.waterMarkImagePacket.Free()
	}	
	s.waterMarkOverlayFilter.Release()
	s.wwaterMarkImageStream.Free()
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

	if m.invDecodeCtx = gmf.NewCodecCtx(m.invCodec); m.invDecodeCtx == nil {
		return fmt.Errorf("unable to create video codec context for %s", m.minfo.UID)
	}

	if err = m.invstream.GetCodecPar().ToContext(m.invDecodeCtx); err != nil {
		return fmt.Errorf("Failed to copy video decoder parameters to input decoder context  for %s", m.minfo.UID)
	}

	if err = m.invDecodeCtx.Open(nil); err != nil {
		return fmt.Errorf("Failed to open decoder for video stream  for %s", m.minfo.UID)
	}

	return err
}

func (s *Streamer) setupInputAudioDecodeCtx(m *Mstream ,adecoderName string) error {

	var err error

	m.inastream, err = m.inctx.GetBestStream(gmf.AVMEDIA_TYPE_AUDIO)
	if err != nil {
		m.inastream.Free()
		log.Printf("No audio stream found in '%s'\n", m.minfo.UID)
		return fmt.Errorf("No audio stream found in '%s'", m.minfo.UID)
	}

	/* audio stream extract and decode context set up */
	m.inaCodec, err = gmf.FindDecoder(adecoderName)
	if err != nil {
		log.Printf("coud not fine audio stream decoder for '%s'\n", m.minfo.UID)
		return fmt.Errorf("coud not fine video stream decoder for '%s'", m.minfo.UID)
	}

	if m.inaDecodeCtx = gmf.NewCodecCtx(m.inaCodec); m.inaDecodeCtx == nil {
		return fmt.Errorf("unable to create audio codec context for %s", m.minfo.UID)
	}

	if err = m.inastream.GetCodecPar().ToContext(m.inaDecodeCtx); err != nil {
		return fmt.Errorf("Failed to copy audio decoder parameters to input decoder context  for %s", m.minfo.UID)
	}

	if err = m.inaDecodeCtx.Open(nil); err != nil {
		return fmt.Errorf("Failed to open decoder for audio stream  for %s", m.minfo.UID)
	}
	return err
}

func (s *Streamer) removeStream(suid string) {

	if s.streamExisted(suid) {

		/* execution order may matters */
		s.mstreams[suid].invDecodeCtx.Close()
		s.mstreams[suid].inaDecodeCtx.Close()
		s.mstreams[suid].invDecodeCtx.Free()
		s.mstreams[suid].inaDecodeCtx.Free()
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

func (s *Streamer) setupOutputVideoEncodeCtx(vencoderName string) error {

	var err error

	s.outvCodec, err = gmf.FindEncoder(vencoderName)
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
		//{Key: "pixel_format", Val: gmf.AV_PIX_FMT_YUV420P},
		{Key: "video_size", Val: s.mstreams[suid].invDecodeCtx.GetVideoSize()},
		{Key: "b", Val: 500000},
	}
	s.outvEncodeCtx.SetPixFmt(s.mstreams[suid].invDecodeCtx.PixFmt())
	s.outvEncodeCtx.SetProfile(gmf.FF_PROFILE_H264_BASELINE)
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

func (s *Streamer) setupOutputAudioEncodeCtxOptions(suid string) error {

	var err error

	s.outaOptions = []gmf.Option{
		{Key: "time_base", Val: gmf.AVR{Num: 1, Den: STREAM_VIDEO_FRAMERATE}},
		{Key: "ar", Val: s.mstreams[suid].inaDecodeCtx.SampleRate()},
		{Key: "ac", Val: s.mstreams[suid].inaDecodeCtx.Channels()},
		{Key: "channel_layout", Val: s.mstreams[suid].inaDecodeCtx.GetDefaultChannelLayout(s.mstreams[suid].inaDecodeCtx.Channels())},
	}
	s.outaEncodeCtx.SetSampleFmt(s.mstreams[suid].inaDecodeCtx.SampleFmt())
	s.outaEncodeCtx.SelectSampleRate()
	s.outaEncodeCtx.SetOptions(s.outaOptions)

	if s.outctx.IsGlobalHeader() {
		s.outaEncodeCtx.SetFlag(gmf.CODEC_FLAG_GLOBAL_HEADER)
	}

	if s.outaCodec.IsExperimental() {
		s.outaEncodeCtx.SetStrictCompliance(gmf.FF_COMPLIANCE_EXPERIMENTAL)
	}

	if err = s.outaEncodeCtx.Open(nil); err != nil {
		return fmt.Errorf("Failed to open encoder for audio stream  for %s", suid)
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

		err = s.initStreamingOutputCtx(STREAMER_PUSH_URL)
		if err != nil {
			return err
		}
		err = s.setupOutputVideoEncodeCtx(VIDEO_ENCODE_CODEC_NAME_X264)
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

		err =s.initWaterMarkWithInputVideoStream(WATERMARK_IMG_URL,suid,WATERMARK_POSITION_TOP_RIGHT)
		if err != nil {
			return err
		}	

		err = s.setupOutputAudioEncodeCtx()
		if err != nil {
			return err
		}

		err = s.setupOutputAudioEncodeCtxOptions(suid)
		if err != nil {
			return err
		}

		err = s.setupOutputAudioStream()
		if err != nil {
			return err
		}

		s.currentStreamingUID = suid
		s.outctx.SetStartTime(0)
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

		for {		

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

				frame, errInt = s.mstreams[suid].inaDecodeCtx.Decode2(pkt)

				if errInt < 0 && gmf.AvErrno(errInt) == syscall.EAGAIN {
					continue
				} else if errInt == gmf.AVERROR_EOF {
					return fmt.Errorf("EOF in audio Decode2, handle it\n")
				} else if errInt < 0 {
					return fmt.Errorf("Unexpected error audio - %s\n", gmf.AvError(errInt))
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

				frame, errInt = s.mstreams[suid].invDecodeCtx.Decode2(pkt)

				if errInt < 0 && gmf.AvErrno(errInt) == syscall.EAGAIN {
					continue
				} else if errInt == gmf.AVERROR_EOF {
					return fmt.Errorf("EOF in video Decode2, handle it\n")
				} else if errInt < 0 {
					return fmt.Errorf("Unexpected error video - %s\n", gmf.AvError(errInt))
				}


				if s.waterMarkOverlayFilter != nil{

					if !filterInitedOnce{

						if err := s.waterMarkOverlayFilter.AddFrame(frame, 0, 0); err != nil {
							return fmt.Errorf("%s\n", err)
						}
						filterInitedOnce=true
	
						if err := s.waterMarkOverlayFilter.AddFrame(s.waterMarkImageFrame, 1, 4); err != nil {
							return fmt.Errorf("%s\n", err)
						}
						s.waterMarkOverlayFilter.RequestOldest()
						s.waterMarkOverlayFilter.Close(1)
	
					}else{
	
						if err := s.waterMarkOverlayFilter.AddFrame(frame, 0, 4); err != nil {
							return fmt.Errorf("%s\n", err)
	
						}
					}
	
					if frames, err = s.waterMarkOverlayFilter.GetFrame(); err != nil && len(frames) == 0 {
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

	if s.waterMarkOverlayFilter != nil{
		s.releaseWaterMarkResource()
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

func (*Streamer) setCurrentStreaming() {

}

func (s *Streamer) rotateStreaming() {

}

func (s *Streamer) replaceAudioStreamingWithMp3() {

}

func (s *Streamer) changeStreamingResolution() {

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
	err = Streamer.addNewStream(Minfo)
	if err != nil {
		fmt.Println(err)
	}

	err = Streamer.startStreaming(Minfo)
	fmt.Println(err)
	Streamer.stopStreaming()
}