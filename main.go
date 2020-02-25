package main

import(
	"log"
	"sync"
	"github.com/3d0c/gmf"
	"fmt"
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
	STREAMER_PUSH_URL="udp://127.0.0.1:1234?pkt_size=1316&buffer_size=65535"

	STREAMER_MP3_REPLACE_URL="/Users/s1ngular/GoWork/src/github.com/organicio/SakuraTears.mp3";
)


type Mstream struct{
	Schema string `json:"schema"`
	Vhost string `json:"vhost"`
	AppName string `json:"app"`
	StreamId string `json:"stream"`
	inastream *gmf.Stream
	invstream *gmf.Stream
	inctx  *gmf.FmtCtx
}

type Streaming struct{
	mstreams map[string]*Mstream
	currentStreamingUID  string
	outctx *gmf.FmtCtx
	outcodec *gmf.Codec
	outcodecCtx  *gmf.CodecCtx
	outoptions []gmf.Option
	outpkt  *gmf.Packet
	outframes  []*gmf.Frame
	mux sync.Mutex
}



func (s *Streaming) addStream(suid string , m Mstream){

	if s.streamExisted(suid) {
		if s.isStreaming() && s.currentStreamingUID == suid {
			s.stopStreaming();
		}
		s.removeStream(suid);
	}

	inputCtx, err := gmf.NewInputCtx(suid)
	if err != nil {
		log.Fatalf("Error creating context - %s\n", err)
	}

	m.inctx=inputCtx;


	invstream, err := inputCtx.GetBestStream(gmf.AVMEDIA_TYPE_VIDEO)
	if err != nil {
		log.Printf("No video stream found in '%s'\n", suid)
		return
	}

	m.invstream=invstream;


	inastream, err := inputCtx.GetBestStream(gmf.AVMEDIA_TYPE_AUDIO)
	if err != nil {
		log.Printf("No audio stream found in '%s'\n", suid)
		return
	}	

	m.inastream=inastream
	s.mstreams[suid]=&m

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

func (s *Streaming) startStreaming(suid string){

	if(s.currentStreamingUID != suid){
		octx, err := gmf.NewOutputCtx(STREAMER_PUSH_URL)
		if err != nil {
			log.Fatalf("fail to create output context for streaming to server '%s'\n", STREAMER_PUSH_URL)
		}
		s.outctx=octx
		err=s.addVideoStreamToOutCtx(VIDEO_CODEC_NAME_X264,s.mstreams[suid].invstream);
		err=s.addAudioStreamToOutCtx(AUDIO_CODEC_NAME_AAC,s.mstreams[suid].inastream);
		defer octx.Free()

	}else{
		fmt.Printf("already streaming this '%s'\n", suid)
	}
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

func (s *Streaming) addVideoStreamToOutCtx(encoderName string, vstream *gmf.Stream) error{

	return nil
}

func (s *Streaming) addAudioStreamToOutCtx(encoderName string, astream *gmf.Stream) error{
	return nil
}

var Streamer=&Streaming{mstreams:make(map[string]*Mstream)}
func main(){
	var m Mstream
	Streamer.addStream("rtmp://202.69.69.180:443/webcast/bshdlive-pc",m);
	Streamer.addStream("rtmp://202.69.69.180:443/webcast/bshdlive-pc",m);
	fmt.Println(len(Streamer.mstreams))
}