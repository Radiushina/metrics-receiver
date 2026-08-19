package proto

import (
	"testing"

	models "github.com/Radiushina/metrics-receiver.git/internal/model"
)

func TestToProtoFromProto_RoundTrip(t *testing.T) {
	g := 1.25
	d := int64(7)
	in := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &g},
		{ID: models.PollCount, MType: models.Counter, Delta: &d},
	}

	got, err := FromProto(ToProto(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].ID != "Alloc" || got[0].MType != models.Gauge || got[0].Value == nil || *got[0].Value != g {
		t.Fatalf("gauge %+v", got[0])
	}
	if got[1].ID != models.PollCount || got[1].MType != models.Counter || got[1].Delta == nil || *got[1].Delta != d {
		t.Fatalf("counter %+v", got[1])
	}
}

func TestFromProto_MissingID(t *testing.T) {
	_, err := FromProto([]*Metric{{Type: Metric_GAUGE, Value: 1}})
	if err == nil {
		t.Fatal("expected error")
	}
}
