//go:build integration

package kafka_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aeswibon/manga-cdc/scraper/internal/kafka"
	"github.com/aeswibon/manga-cdc/scraper/internal/model"
	segkafka "github.com/segmentio/kafka-go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

const (
	redpandaPort = "29092/tcp"
	brokerAddr   = "127.0.0.1:29092"
)

func TestIntegration_ProducerPublishAndConsumerRead(t *testing.T) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image: "redpandadata/redpanda:v24.2.7",
		Cmd: []string{
			"redpanda", "start",
			"--smp", "1",
			"--memory", "256M",
			"--overprovisioned",
			"--node-id", "0",
			"--kafka-addr", "PLAINTEXT://0.0.0.0:29092",
			"--advertise-kafka-addr", "PLAINTEXT://localhost:29092",
		},
		ExposedPorts: []string{redpandaPort},
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.PortBindings = nat.PortMap{
				nat.Port(redpandaPort): []nat.PortBinding{
					{HostIP: "127.0.0.1", HostPort: "29092"},
				},
			}
		},
		WaitingFor: wait.ForLog("Successfully started Redpanda").WithStartupTimeout(60 * time.Second),
	}

	redpandaContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start redpanda: %v", err)
	}
	defer redpandaContainer.Terminate(ctx)

	brokers := brokerAddr

	code, _, err := redpandaContainer.Exec(ctx, []string{
		"rpk", "topic", "create", "test-integration-chapters",
	})
	if err != nil || code != 0 {
		t.Fatalf("create topic (exit=%d): %v", code, err)
	}

	producer, err := kafka.NewProducer(brokers, "test-integration-chapters", "", "")
	if err != nil {
		t.Fatalf("create producer: %v", err)
	}
	defer producer.Close()

	chapter := model.Chapter{
		ID:       "int-ch-1",
		SeriesID: "int-s-1",
		Number:   1,
		Title:    "Integration Chapter",
		URL:      "https://test.com/int-ch-1",
		IsNew:    true,
	}

	err = producer.PublishChapterEvent(ctx, chapter)
	if err != nil {
		t.Fatalf("publish chapter event: %v", err)
	}

	reader := segkafka.NewReader(segkafka.ReaderConfig{
		Brokers:   []string{brokers},
		Topic:     "test-integration-chapters",
		GroupID:   "test-integration-group",
		MinBytes:  1,
		MaxBytes:  10e6,
		MaxWait:   5 * time.Second,
	})
	defer reader.Close()

	msg, err := reader.ReadMessage(ctx)
	if err != nil {
		t.Fatalf("read message: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}

	op, _ := payload["op"].(string)
	if op != "c" {
		t.Errorf("expected op=c, got %s", op)
	}

	after, _ := payload["after"].(map[string]any)
	if after["id"] != "int-ch-1" {
		t.Errorf("expected after.id=int-ch-1, got %v", after["id"])
	}
	if after["series_id"] != "int-s-1" {
		t.Errorf("expected after.series_id=int-s-1, got %v", after["series_id"])
	}
}
