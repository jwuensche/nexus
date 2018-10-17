package metrics

import (
	"net/http/httptest"
	"testing"
)

func TestServer(t *testing.T) {
	Init(3444, true, false)
	mp := GetSubMapOf(MetricGlobal.mp, "tester")
	IncrementAtomicUint64KeyOf(mp, "test")
	IncreaseAtomicUint64KeyOf(mp, "test", 54)
	r := httptest.NewRequest("", "localhost:3444", nil)
	w := httptest.NewRecorder()
	metricToJSON(w, r)
}

func TestFlat(t *testing.T) {
	IncrementAtomicUint64KeyOf(MetricGlobal.mp, "testing")
	IncreaseAtomicUint64KeyOf(MetricGlobal.mp, "testing", 54)
}

func TestMap(t *testing.T) {
	mp := GetSubMapOf(MetricGlobal.mp, "tester")
	IncrementAtomicUint64KeyOf(mp, "test")
	IncreaseAtomicUint64KeyOf(mp, "test", 54)
}

func TestBuiltinHandlers(t *testing.T) {
	RecvMsgCountHandler()
	SendMsgCountHandler()
	SendMsgLenHandler(54)
	RecvMsgLenHandler(54)
}

func TestConvert(t *testing.T) {
	processMtr(MetricGlobal.mp)
}