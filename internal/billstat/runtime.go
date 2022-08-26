package billstat

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardDNS/internal/agd"
	"github.com/AdguardTeam/AdGuardDNS/internal/metrics"
	"github.com/AdguardTeam/golibs/log"
)

// Runtime Billing Statistics Recorder

// RuntimeRecorderConfig is the configuration structure for a runtime billing
// statistics recorder.  All fields must be non-empty.
type RuntimeRecorderConfig struct {
	// Uploader is used to upload the billing statistics records to.
	Uploader Uploader
}

// NewRuntimeRecorder creates a new runtime billing statistics database.  c must
// be non-nil.
func NewRuntimeRecorder(c *RuntimeRecorderConfig) (r *RuntimeRecorder) {
	return &RuntimeRecorder{
		mu:       &sync.Mutex{},
		records:  Records{},
		uploader: c.Uploader,
	}
}

// RuntimeRecorder is the runtime billing statistics recorder.  The records kept
// here are not persistent.
type RuntimeRecorder struct {
	// mu protects records and syncTime.
	mu *sync.Mutex

	// records are the statistics records awaiting their synchronization.
	records Records

	// uploader is the uploader to which the billing statistics records are
	// uploaded.
	uploader Uploader
}

// type check
var _ Recorder = (*RuntimeRecorder)(nil)

// Record implements the Recorder interface for *RuntimeRecorder.
func (r *RuntimeRecorder) Record(
	ctx context.Context,
	id agd.DeviceID,
	ctry agd.Country,
	asn agd.ASN,
	start time.Time,
	proto agd.Protocol,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec := r.records[id]
	if rec == nil {
		r.records[id] = &Record{
			Time:    start,
			Country: ctry,
			ASN:     asn,
			Queries: 1,
			Proto:   proto,
		}

		metrics.BillStatBufSize.Add(1)
	} else {
		rec.Time = start
		rec.Country = ctry
		rec.ASN = asn
		rec.Queries++
		rec.Proto = proto
	}
}

// type check
var _ agd.Refresher = (*RuntimeRecorder)(nil)

// Refresh implements the agd.Refresher interface for *RuntimeRecorder.  It
// uploads the currently available data and resets it.
func (r *RuntimeRecorder) Refresh(ctx context.Context) (err error) {
	records := r.resetRecords()
	defer func() {
		if err != nil {
			r.remergeRecords(records)
			log.Info("billstat: refresh failed, records remerged")
		}

		metrics.SetStatusGauge(metrics.BillStatUploadStatus, err)
	}()

	err = r.uploader.Upload(ctx, records)
	if err != nil {
		return fmt.Errorf("uploading billstat records: %w", err)
	}

	metrics.BillStatUploadTimestamp.SetToCurrentTime()

	return nil
}

// resetRecords returns the current data and resets the records map to an empty
// map.
func (r *RuntimeRecorder) resetRecords() (records Records) {
	r.mu.Lock()
	defer r.mu.Unlock()

	records, r.records = r.records, Records{}

	metrics.BillStatBufSize.Set(0)

	return records
}

// remergeRecords merges records back into the database, unless there is already
// a newer record, in which case it merges the results.
func (r *RuntimeRecorder) remergeRecords(records Records) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for devID, prev := range records {
		if curr, ok := r.records[devID]; !ok {
			r.records[devID] = prev
		} else {
			curr.Queries += prev.Queries
		}
	}

	metrics.BillStatBufSize.Set(float64(len(r.records)))
}
