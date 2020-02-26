package main

import(
	"log"
	"sync"
	"github.com/3d0c/gmf"
	"fmt"
	"io"
)

const (
	HW_VIDEO_CODEC_NAME_H264_MMAL="h264_mmal"
	HW_VIDEO_CODEC_NAME_H264_OMX="h264_omx"
	HW_VIDEO_CODEC_NAME_H264_V4L2M2M="h264_v4l2m2m"
	VIDEO_CODEC_NAME_X264="libx264"

	AUDIO_CODEC_NAME_AAC="aac"
	AUDIO_CODEC_NAME_MP3="mp3"
)

var (

	STREAM_RESOLUTION_HIGHT=21
	STREAM_RESOLUTION_MEDIUM=28
	STREAM_RESOLUTION_LOW=34

	STREAM_MAX_BITRATE="1M"
	STREAM_BUFSIZE="2M"
	STREAM_AUDIO_BITRATE="64K"
	STREAM_VIDEO_BITRATE="900K"
	STREAM_VIDEO_FRAMETATE=25
	
	STREAMER_UUID="sdfrweqrewqr0943rasdfds"
	STREAMER_SECRET="FDQER345435435"
	STREAMER_PUSH_URL="/Users/s1ngular/GoWork/src/github.com/organicio/tesdtt.mp4"

	STREAMER_MP3_REPLACE_URL="/Users/s1ngular/GoWork/src/github.com/organicio/SakuraTears.mp3";
)


type Mstream struct{
	Schema string `json:"schema"`
	Vhost string `json:"vhost"`
	AppName string `json:"app"`
	StreamId string `json:"stream"`
	inctx  *gmf.FmtCtx
	inastream *gmf.Stream
	invstream *gmf.Stream
	inaCodec *gmf.Codec
	invCodec *gmf.Codec
	inaDecodecCtx *gmf.CodecCtx
	invDecodecCtx *gmf.CodecCtx


	pkt    *gmf.Packet
	frames []*gmf.Frame
}

type Streaming struct{
	mstreams map[string]*Mstream
	currentStreamingUID  string

	outctx *gmf.FmtCtx
	outvstream *gmf.Stream
	outastream *gmf.Stream
	outvCodec *gmf.Codec
	outaCodec *gmf.Codec
	outvEncodeCtx *gmf.CodecCtx
	outaEncodeCtx *gmf.CodecCtx
	outvOptions []gmf.Option
	outaOptions []gmf.Option
	mux sync.Mutex
}



func (s *Streaming) addStream(suid string , m Mstream) error{

	if s.streamExisted(suid) {
		if s.isStreaming() && s.currentStreamingUID == suid {
			s.stopStreaming();
		}
		s.removeStream(suid);
	}

	var err error

	m.inctx, err = gmf.NewInputCtx(suid)
	if err != nil {
		m.inctx.Free()
		log.Printf("Error creating context for '%s' - %s\n", suid,err)
		return fmt.Errorf("Error creating context for '%s' - %s", suid,err)
	}


	/* video stream extract and decode context set up */
	m.invstream, err = m.inctx.GetBestStream(gmf.AVMEDIA_TYPE_VIDEO)
	if err != nil {
		m.invstream.Free();
		log.Printf("No video stream found in '%s'\n", suid)
		return fmt.Errorf("No video stream found in '%s'", suid)
	}

	m.invCodec ,err=gmf.FindDecoder(m.invstream.GetCodecPar().GetCodecId())
	if(err != nil){
		log.Printf("coud not fine video stream decoder for '%s'\n", suid)
		return fmt.Errorf("coud not fine video stream decoder for '%s'", suid)		
	}

	if m.invDecodecCtx=gmf.NewCodecCtx(m.invCodec);m.invDecodecCtx == nil{
		return fmt.Errorf("unable to create video codec context for %s",suid)
	}

	if err = m.invstream.GetCodecPar().ToContext(m.invDecodecCtx);err !=nil{
		return fmt.Errorf("Failed to copy video decoder parameters to input decoder context  for %s",suid)
	}


	if err=m.invDecodecCtx.Open(nil);err != nil{
		return fmt.Errorf("Failed to open decoder for video stream  for %s",suid)
	}


	/* audio stream extract and decode context set up */
	m.inastream, err = m.inctx.GetBestStream(gmf.AVMEDIA_TYPE_AUDIO)
	if err != nil {
		m.inastream.Free()
		log.Printf("No audio stream found in '%s'\n", suid)
		return fmt.Errorf("No audio stream found in '%s'", suid)
	}	
	
	m.inaCodec ,err=gmf.FindDecoder(m.inastream.GetCodecPar().GetCodecId())
	if(err != nil){
		log.Printf("coud not fine audio stream decoder for '%s'\n", suid)
		return fmt.Errorf("coud not fine video stream decoder for '%s'", suid)		
	}	

	if m.inaDecodecCtx=gmf.NewCodecCtx(m.inaCodec);m.inaDecodecCtx == nil{
		return fmt.Errorf("unable to create audio codec context for %s",suid)
	}

	if err = m.inastream.GetCodecPar().ToContext(m.inaDecodecCtx);err !=nil{
		return fmt.Errorf("Failed to copy audio decoder parameters to input decoder context  for %s",suid)
	}

	if err=m.inaDecodecCtx.Open(nil);err != nil{
		return fmt.Errorf("Failed to open decoder for audio stream  for %s",suid)
	}

	m.inctx.Dump()

	s.mstreams[suid]=&m
	return nil

}


func (s *Streaming) removeStream(suid string){
	//free up resource , or cause memory leaks

	delete(s.mstreams, suid)
}

func (s *Streaming) streamExisted(suid string) bool{

	if _, ok := s.mstreams[suid];ok {
		return true
	}
	return false
}

func (s *Streaming) isStreaming() bool {

	return len(s.currentStreamingUID) > 0
}

func (s *Streaming) startStreaming(suid string) error{

	var err error
	if(s.currentStreamingUID != suid){

		s.outctx, err = gmf.NewOutputCtx(STREAMER_PUSH_URL)
		if err != nil {
			s.outctx.Free()
			log.Printf("fail to create output context for streaming to server '%s' '%s' \n", STREAMER_PUSH_URL,err)
			return fmt.Errorf("fail to create output context for streaming to server '%s' '%s'", STREAMER_PUSH_URL,err)
		}

		/* out video codec and stream context init*/
		/* we choose transcoding to same codec */
		
		s.outvCodec,err=gmf.FindEncoder(s.mstreams[suid].invDecodecCtx.Id())
		if(err != nil){
			return fmt.Errorf("video encoder not found: '%s'",err)
		}

		if s.outvEncodeCtx=gmf.NewCodecCtx(s.outvCodec);s.outvEncodeCtx == nil{
			return fmt.Errorf("create video encoder context failed: '%s'",err)
		}

		s.outvOptions = append(
			[]gmf.Option{
				{Key: "time_base", Val: gmf.AVR{Num: 1, Den: 25}},
				{Key: "pixel_format", Val: gmf.AV_PIX_FMT_YUV420P},
				// Save original
				{Key: "video_size", Val: s.mstreams[suid].invDecodecCtx.GetVideoSize()},
				{Key: "b", Val: 500000},
			},
		)

		s.outvEncodeCtx.SetProfile(s.mstreams[suid].invDecodecCtx.GetProfile())
		s.outvEncodeCtx.SetOptions(s.outvOptions)

		if s.outctx.IsGlobalHeader() {
			s.outvEncodeCtx.SetFlag(gmf.CODEC_FLAG_GLOBAL_HEADER)
		}
	
		if s.outvCodec.IsExperimental() {
			s.outvEncodeCtx.SetStrictCompliance(gmf.FF_COMPLIANCE_EXPERIMENTAL)
		}

		if err=s.outvEncodeCtx.Open(nil);err != nil{
			return fmt.Errorf("Failed to open encoder for video stream  for %s",suid)
		}

		if s.outvstream=s.outctx.NewStream(s.outvCodec);s.outvstream == nil{
			return fmt.Errorf("unable to create new video stream in output context: '%s'",err)
		}

		if s.outvstream.GetCodecPar().FromContext(s.outvEncodeCtx);err != nil{

			return fmt.Errorf("Failed to copy video encoder parameters to output stream - %s", err)
		}

		s.outvstream.SetTimeBase(gmf.AVR{Num: 1, Den: 25})
		s.outvstream.SetRFrameRate(gmf.AVR{Num: 25, Den: 1})



		/* out audio codec and stream context init*/
		s.outaCodec,err=gmf.FindEncoder(s.mstreams[suid].inaDecodecCtx.Id())
		if(err != nil){
			return fmt.Errorf("audio encoder not found: '%s'",err)
		}

		if s.outaEncodeCtx=gmf.NewCodecCtx(s.outaCodec);s.outaEncodeCtx == nil{
			return fmt.Errorf("create audio encoder context failed: '%s'",err)
		}

		s.outaOptions = append(
			[]gmf.Option{
				{Key: "time_base", Val: s.mstreams[suid].inaDecodecCtx.TimeBase().AVR()},
				{Key: "ar", Val: s.mstreams[suid].inaDecodecCtx.SampleRate()},
				{Key: "ac", Val:s.mstreams[suid].inaDecodecCtx.Channels()},
				{Key: "channel_layout", Val: s.outaEncodeCtx.SelectChannelLayout()},
			},
		)
		s.outaEncodeCtx.SetSampleFmt(s.mstreams[suid].inaDecodecCtx.SampleFmt())
		s.outaEncodeCtx.SelectSampleRate()
		s.outaEncodeCtx.SetOptions(s.outaOptions)

		if s.outctx.IsGlobalHeader() {
			s.outaEncodeCtx.SetFlag(gmf.CODEC_FLAG_GLOBAL_HEADER)
		}
	
		if s.outaCodec.IsExperimental() {
			s.outaEncodeCtx.SetStrictCompliance(gmf.FF_COMPLIANCE_EXPERIMENTAL)
		}

		if err=s.outaEncodeCtx.Open(nil);err != nil{
			return fmt.Errorf("Failed to open encoder for audio stream  for %s",suid)
		}

		if s.outastream=s.outctx.NewStream(s.outaCodec);s.outastream == nil{
			return fmt.Errorf("unable to create new audio stream in output context: '%s'",err)
		}

		if s.outastream.GetCodecPar().FromContext(s.outaEncodeCtx);err != nil{

			return fmt.Errorf("Failed to copy audio encoder parameters to output stream - %s", err)
		}

		s.currentStreamingUID=suid;

		if err := s.outctx.WriteHeader(); err != nil {
			return fmt.Errorf("error writing header - %s", err)
		}


		var (
			pkt   *gmf.Packet
			i int
			streamIdx int
			pts  int64       = 0
			flush     int = -1
		)

		for i=0;i< 1000;i++ {

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

			if streamIdx == s.mstreams[suid].inastream.Index(){
				frames, err = s.mstreams[suid].inaDecodecCtx.Decode(pkt)
				if err != nil {
					return fmt.Errorf("error decoding - %s", err)
				}
				fmt.Printf("decoded audio stream index : %d",streamIdx)
				for _, frame := range frames {
					frame.SetPts(pts)
					pts++
				}
				packets, err := s.outaEncodeCtx.Encode(frames, flush)
				for _, op := range packets {
					gmf.RescaleTs(op, s.outaEncodeCtx.TimeBase(), s.outastream.TimeBase())
					op.SetStreamIndex(s.outastream.Index())
	
					if err = s.outctx.WritePacket(op); err != nil {
						break
					}
	
					op.Free()
				}
	
			}
			if streamIdx == s.mstreams[suid].invstream.Index(){
				frames, err = s.mstreams[suid].invDecodecCtx.Decode(pkt)
				if err != nil {
					return fmt.Errorf("error decoding - %s", err)
				}
				fmt.Printf("decoded video stream index : %d",streamIdx)
				for _, frame := range frames {
					frame.SetPts(pts)
					pts++
				}

				packets, err := s.outvEncodeCtx.Encode(frames, flush)
				for _, op := range packets {
					gmf.RescaleTs(op, s.outvEncodeCtx.TimeBase(), s.outvstream.TimeBase())
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
		
	}else{
		fmt.Printf("already streaming this '%s'\n", suid)
	}
	return nil
}

func (s *Streaming) stopStreaming(){

}

func (*Streaming) setCurrentStreaming(){

}

func (s *Streaming) rotateStreaming(){

}

func (s *Streaming) replaceAudioStreamingWithMp3(){

}

func (s *Streaming) changeStreamingResolution(){

}

var Streamer=&Streaming{mstreams:make(map[string]*Mstream)}
func main(){
	var m Mstream
	var err error
	err=Streamer.addStream("rtmp://mobliestream.c3tv.com:554/live/goodtv.sdp",m);
	//err=Streamer.addStream("/Users/s1ngular/GoWork/src/github.com/organicio/bbb.mp4",m);
	if(err != nil){
		fmt.Println("add stream failed")
	}
	err=Streamer.startStreaming("rtmp://mobliestream.c3tv.com:554/live/goodtv.sdp")
	fmt.Println(err)
}