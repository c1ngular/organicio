package main

import (
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/3d0c/gmf"
)

const (
	HW_VIDEO_CODEC_NAME_H264_MMAL    = "h264_mmal"
	HW_VIDEO_CODEC_NAME_H264_OMX     = "h264_omx"
	HW_VIDEO_CODEC_NAME_H264_V4L2M2M = "h264_v4l2m2m"
	VIDEO_CODEC_ENCODER_NAME_X264    = "libx264"
	VIDEO_CODEC_DECODER_NAME_X264    = "h264"

	AUDIO_CODEC_NAME_AAC = "aac"
	AUDIO_CODEC_NAME_MP3 = "mp3"
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

	STREAMER_MP3_REPLACE_URL = "/Users/s1ngular/GoWork/src/github.com/organicio/SakuraTears.mp3"
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
	mux           sync.Mutex
}

func (s *Streamer) addStream(mInfo StreamInfo) error {

	var err error

	m := &Mstream{minfo: mInfo}

	if !s.streamExisted(m.minfo.UID) {
		m.inctx, err = gmf.NewInputCtx(m.minfo.UID)
		if err != nil {
			m.inctx.Free()
			log.Printf("Error creating context for '%s' - %s\n", m.minfo.UID, err)
			return fmt.Errorf("Error creating context for '%s' - %s", m.minfo.UID, err)
		}

		err = s.setupInputVideoDecodeCtx(m, VIDEO_CODEC_DECODER_NAME_X264)
		if err != nil {
			return fmt.Errorf("failed to setup input video decoder '%s'", m.minfo.UID)
		}

		err = s.setupInputAudioDecodeCtx(m, AUDIO_CODEC_NAME_AAC)
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

func (s *Streamer) setupInputAudioDecodeCtx(m *Mstream, adecoderName string) error {

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
		{Key: "video_size", Val: s.mstreams[suid].invDecodecCtx.GetVideoSize()},
		{Key: "b", Val: 500000},
	}
	s.outvEncodeCtx.SetPixFmt(s.mstreams[suid].invDecodecCtx.PixFmt())
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

func (s *Streamer) setupOutputAudioEncodeCtx(aencoderName string) error {

	var err error

	s.outaCodec, err = gmf.FindEncoder(aencoderName)
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
		{Key: "ar", Val: s.mstreams[suid].inaDecodecCtx.SampleRate()},
		{Key: "ac", Val: s.mstreams[suid].inaDecodecCtx.Channels()},
		{Key: "channel_layout", Val: s.mstreams[suid].inaDecodecCtx.GetDefaultChannelLayout(s.mstreams[suid].inaDecodecCtx.Channels())},
	}
	s.outaEncodeCtx.SetSampleFmt(s.mstreams[suid].inaDecodecCtx.SampleFmt())
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
		err = s.setupOutputVideoEncodeCtx(VIDEO_CODEC_ENCODER_NAME_X264)
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

		err = s.setupOutputAudioEncodeCtx(AUDIO_CODEC_NAME_AAC)
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

		if err := s.outctx.WriteHeader(); err != nil {
			return fmt.Errorf("error writing header - %s", err)
		}

		var (
			pkt       *gmf.Packet
			i         int
			streamIdx int
			flush     int = -1
		)

		for i = 0; i < 10000; i++ {

			if flush < 0 {
				pkt, err = s.mstreams[suid].inctx.GetNextPacket()
				if err != nil && err != io.EOF {
					if pkt != nil {
						pkt.Free()
					}
					log.Fatalf("error getting next packet - %s", err)
				} else if err != nil && pkt == nil {
					log.Printf("=== flushing \n")
					flush++
					break
				}
			}

			if flush < 0 {
				streamIdx = pkt.StreamIndex()
			} else {
				streamIdx = flush
				flush++
			}

			var frames []*gmf.Frame

			if streamIdx == s.mstreams[suid].inastream.Index() {
				frames, err = s.mstreams[suid].inaDecodecCtx.Decode(pkt)
				if err != nil {
					return fmt.Errorf("error decoding - %s", err)
				}

				packets, err := s.outaEncodeCtx.Encode(frames, flush)
				for _, op := range packets {

					gmf.RescaleTs(op, s.mstreams[suid].inastream.TimeBase(), s.outastream.TimeBase())
					op.SetStreamIndex(s.outastream.Index())
					if err = s.outctx.WritePacket(op); err != nil {
						break
					}

					op.Free()
				}

			}

			if streamIdx == s.mstreams[suid].invstream.Index() {
				frames, err = s.mstreams[suid].invDecodecCtx.Decode(pkt)
				if err != nil {
					return fmt.Errorf("error decoding - %s", err)
				}

				packets, err := s.outvEncodeCtx.Encode(frames, flush)
				for _, op := range packets {

					gmf.RescaleTs(op, s.mstreams[suid].invstream.TimeBase(), s.outvstream.TimeBase())
					op.SetStreamIndex(s.outvstream.Index())
					if err = s.outctx.WritePacket(op); err != nil {
						break
					}

					op.Free()
				}

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
	err = Streamer.addStream(Minfo)
	if err != nil {
		fmt.Println("/Users/s1ngular/GoWork/src/github.com/organicio/bbb.mp4")
	}

	err = Streamer.startStreaming(Minfo)
	fmt.Println(err)
	Streamer.stopStreaming()
}
