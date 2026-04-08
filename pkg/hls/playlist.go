package hls

import (
	"fmt"
	"math"
	"net/http"
)

func GenerateVODPlaylist(sess *Session, endpointPrefix string) string {
	totalSecs := float64(sess.DurationTicks) / 10000000.0
	segLen := float64(sess.SegmentLength)

	numWholeSegments := int(totalSecs / segLen)
	remainder := totalSecs - float64(numWholeSegments)*segLen

	totalSegments := numWholeSegments
	if remainder > 0.1 {
		totalSegments++
	}

	targetDuration := int(math.Ceil(segLen))

	result := "#EXTM3U\n"
	result += "#EXT-X-VERSION:3\n"
	result += fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", targetDuration)
	result += "#EXT-X-MEDIA-SEQUENCE:0\n"
	result += "#EXT-X-PLAYLIST-TYPE:VOD\n"

	var currentTicks int64
	for i := 0; i < totalSegments; i++ {
		dur := segLen
		if i == totalSegments-1 && remainder > 0.1 {
			dur = remainder
		}
		lengthTicks := int64(dur * 10000000)

		result += fmt.Sprintf("#EXTINF:%.6f,\n", dur)
		result += fmt.Sprintf("%s%d.ts?runtimeTicks=%d&actualSegmentLengthTicks=%d\n",
			endpointPrefix, i, currentTicks, lengthTicks)

		currentTicks += lengthTicks
	}

	result += "#EXT-X-ENDLIST\n"
	return result
}

func GenerateLivePlaylist(sess *Session) string {
	result := "#EXTM3U\n"
	result += "#EXT-X-VERSION:3\n"
	result += fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", sess.SegmentLength)
	result += "#EXT-X-MEDIA-SEQUENCE:0\n"

	current := sess.CurrentTranscodeIndex()
	if current < 0 {
		return result
	}

	for i := 0; i <= current; i++ {
		result += fmt.Sprintf("#EXTINF:%d.000000,\n", sess.SegmentLength)
		result += fmt.Sprintf("%s%d.ts\n", sess.ID, i)
	}

	return result
}

func ServeMasterPlaylist(w http.ResponseWriter, sess *Session, baseURL string) {
	w.Header().Set("Content-Type", "application/x-mpegURL")
	w.Header().Set("Cache-Control", "no-cache, no-store")

	bandwidth := 10000000
	fmt.Fprintln(w, "#EXTM3U")
	fmt.Fprintf(w, "#EXT-X-STREAM-INF:BANDWIDTH=%d\n", bandwidth)

	if sess.IsLive {
		fmt.Fprintf(w, "%s/Videos/%s/live.m3u8\n", baseURL, sess.ID)
	} else {
		fmt.Fprintf(w, "%s/Videos/%s/main.m3u8\n", baseURL, sess.ID)
	}
}

func ServeMediaPlaylist(w http.ResponseWriter, sess *Session) {
	w.Header().Set("Content-Type", "application/x-mpegURL")
	w.Header().Set("Cache-Control", "no-cache, no-store")

	var playlist string
	if sess.DurationTicks > 0 && !sess.IsLive {
		endpointPrefix := fmt.Sprintf("hls1/main/%s", sess.ID)
		playlist = GenerateVODPlaylist(sess, endpointPrefix)
	} else {
		playlist = GenerateLivePlaylist(sess)
	}

	w.Write([]byte(playlist))
}

func ServeSegment(w http.ResponseWriter, r *http.Request, segPath string) {
	w.Header().Set("Content-Type", "video/mp2t")
	http.ServeFile(w, r, segPath)
}
