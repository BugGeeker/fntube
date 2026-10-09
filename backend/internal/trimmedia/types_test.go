package trimmedia

import (
	"encoding/json"
	"testing"
)

func TestStreamListResultUnmarshalSubtitleStreams(t *testing.T) {
	payload := []byte(`{
		"subtitle_streams": [{
			"media_guid": "a2c26a4753a44caba7c933b676489aa3",
			"title": "",
			"guid": "23904843b1834d218c9a6805dda4f1c2",
			"codec_name": "subrip",
			"codec_type": "subtitle",
			"language": "zh",
			"forced": 0,
			"index": 0,
			"is_default": 0,
			"is_external": 1,
			"format": "srt",
			"trim_id": "",
			"source_id": "",
			"Source": "",
			"create_time": 1790749037372,
			"update_time": 1790749037372,
			"extra_file": 1,
			"is_bitmap": 0,
			"file_size": 0
		}]
	}`)

	var result StreamListResult
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("unmarshal stream list: %v", err)
	}
	if len(result.SubtitleStreams) != 1 {
		t.Fatalf("subtitle stream count = %d, want 1", len(result.SubtitleStreams))
	}
	subtitle := result.SubtitleStreams[0]
	if subtitle.GUID != "23904843b1834d218c9a6805dda4f1c2" {
		t.Fatalf("guid = %q", subtitle.GUID)
	}
	if subtitle.CodecName != "subrip" || subtitle.Language != "zh" || subtitle.Format != "srt" {
		t.Fatalf("unexpected subtitle stream: %+v", subtitle)
	}
	if subtitle.IsExternal != 1 || subtitle.ExtraFile != 1 {
		t.Fatalf("unexpected subtitle flags: %+v", subtitle)
	}
}
