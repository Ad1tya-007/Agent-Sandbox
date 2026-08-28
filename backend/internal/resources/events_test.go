package resources

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestTranslateEventReasons(t *testing.T) {
	ts := metav1.NewTime(time.Date(2026, 8, 27, 20, 1, 2, 0, time.UTC))
	tests := []struct {
		name   string
		ev     corev1.Event
		title  string
		detail string
	}{
		{
			name: "sandbox created",
			ev: corev1.Event{
				Reason:         "Created",
				Message:        "sandbox created",
				InvolvedObject: corev1.ObjectReference{Kind: "Sandbox", Name: "demo"},
			},
			title:  "Sandbox Created",
			detail: "API server accepted the Sandbox resource",
		},
		{
			name: "successful create",
			ev: corev1.Event{
				Reason:         "SuccessfulCreate",
				Message:        "created pod demo",
				InvolvedObject: corev1.ObjectReference{Kind: "Sandbox", Name: "demo"},
			},
			title:  "Sandbox Created",
			detail: "controller created child",
		},
		{
			name: "scheduled",
			ev: corev1.Event{
				Reason:  "Scheduled",
				Message: "Successfully assigned default/demo to worker-1",
			},
			title:  "Scheduled",
			detail: "Assigned to worker-1",
		},
		{
			name: "pulling",
			ev: corev1.Event{
				Reason:  "Pulling",
				Message: `Pulling image "python:3.12-slim"`,
			},
			title:  "Image Pulled",
			detail: "python:3.12-slim pulled",
		},
		{
			name: "pulled",
			ev: corev1.Event{
				Reason:  "Pulled",
				Message: `Successfully pulled image "python:3.12-slim"`,
			},
			title:  "Image Pulled",
			detail: "python:3.12-slim pulled",
		},
		{
			name: "pod created",
			ev: corev1.Event{
				Reason:         "Created",
				Message:        "Created container sandbox",
				InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "demo"},
			},
			title:  "Container Started",
			detail: "sandbox container started",
		},
		{
			name: "started",
			ev: corev1.Event{
				Reason:         "Started",
				Message:        "Started container sandbox",
				InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "demo"},
			},
			title:  "Container Started",
			detail: "sandbox container started",
		},
		{
			name: "killing",
			ev: corev1.Event{
				Reason:  "Killing",
				Message: "Stopping container sandbox",
			},
			title:  "Restarted",
			detail: "Stopping container sandbox",
		},
		{
			name: "backoff",
			ev: corev1.Event{
				Reason:  "BackOff",
				Message: "Back-off restarting failed container sandbox",
			},
			title:  "Restarted",
			detail: "Back-off restarting failed container sandbox",
		},
		{
			name: "unhealthy",
			ev: corev1.Event{
				Reason:  "Unhealthy",
				Message: "Readiness probe failed",
			},
			title:  "Restarted",
			detail: "Readiness probe failed",
		},
		{
			name: "failed",
			ev: corev1.Event{
				Reason:  "Failed",
				Message: "Error: image pull failed",
			},
			title:  "Failed",
			detail: "Error: image pull failed",
		},
		{
			name: "failed scheduling",
			ev: corev1.Event{
				Reason:  "FailedScheduling",
				Message: "0/1 nodes are available: insufficient cpu",
			},
			title:  "Pending",
			detail: "0/1 nodes are available: insufficient cpu",
		},
		{
			name: "default reason",
			ev: corev1.Event{
				Reason:  "Sync",
				Message: "synced",
			},
			title:  "Sync",
			detail: "synced",
		},
		{
			name: "empty reason",
			ev: corev1.Event{
				Message: "something happened",
			},
			title:  "Event",
			detail: "something happened",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ev.UID = types.UID("uid-" + tt.name)
			tt.ev.LastTimestamp = ts
			got := TranslateEvent(tt.ev)
			if got.Title != tt.title || got.Detail != tt.detail {
				t.Fatalf("title/detail = %q %q, want %q %q", got.Title, got.Detail, tt.title, tt.detail)
			}
			if got.ID != "uid-"+tt.name {
				t.Fatalf("id = %q", got.ID)
			}
			if got.At != "2026-08-27T20:01:02Z" {
				t.Fatalf("at = %q", got.At)
			}
		})
	}
}

func TestTranslateEventTimeFallback(t *testing.T) {
	eventTime := metav1.NewMicroTime(time.Date(2026, 8, 27, 21, 0, 0, 0, time.UTC))
	got := TranslateEvent(corev1.Event{
		ObjectMeta: metav1.ObjectMeta{UID: "e1", Name: "demo.1"},
		EventTime:  eventTime,
		Reason:     "Sync",
	})
	if got.At != "2026-08-27T21:00:00Z" {
		t.Fatalf("at = %q", got.At)
	}
}

func TestTranslateEventsDedupAndSort(t *testing.T) {
	t1 := metav1.NewTime(time.Date(2026, 8, 27, 20, 0, 0, 0, time.UTC))
	t2 := metav1.NewTime(time.Date(2026, 8, 27, 20, 1, 0, 0, time.UTC))
	events := []corev1.Event{
		{
			ObjectMeta:     metav1.ObjectMeta{UID: "later", Name: "b"},
			Reason:         "Started",
			LastTimestamp:  t2,
			InvolvedObject: corev1.ObjectReference{Kind: "Pod"},
		},
		{
			ObjectMeta:    metav1.ObjectMeta{UID: "first", Name: "a"},
			Reason:        "Created",
			LastTimestamp: t1,
		},
		{
			ObjectMeta:    metav1.ObjectMeta{UID: "first", Name: "a-dup"},
			Reason:        "Created",
			LastTimestamp: t1,
		},
	}
	got := TranslateEvents(events)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %#v", len(got), got)
	}
	if got[0].ID != "first" || got[0].Title != "Sandbox Created" {
		t.Fatalf("first = %+v", got[0])
	}
	if got[1].ID != "later" || got[1].Title != "Container Started" {
		t.Fatalf("second = %+v", got[1])
	}

	if got := TranslateEvents(nil); got == nil || len(got) != 0 {
		t.Fatalf("nil events = %#v", got)
	}
}

func TestTranslateEventIDWithoutUID(t *testing.T) {
	ts := metav1.NewTime(time.Date(2026, 8, 27, 20, 1, 2, 0, time.UTC))
	got := TranslateEvent(corev1.Event{
		ObjectMeta:    metav1.ObjectMeta{Name: "demo.123"},
		Reason:        "Sync",
		LastTimestamp: ts,
	})
	if got.ID != "demo.123|2026-08-27T20:01:02Z" {
		t.Fatalf("id = %q", got.ID)
	}
}
