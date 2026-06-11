package presets

import "fmt"

type Hints struct {
	DatabaseComment string
	KafkaComment    string
	QStashComment   string
	SetupNotes      []string
}

var database = map[string]Hints{
	"generic": {},
	"aiven": {
		DatabaseComment: "# Aiven Postgres — use sslmode=require and credentials from the Aiven console",
		SetupNotes:      []string{"Create a PostgreSQL service in the [Aiven console](https://console.aiven.io/)."},
	},
	"neon": {
		DatabaseComment: "# Neon — use sslmode=require; prefer the direct (non-pooled) connection for migrations",
		SetupNotes:      []string{"Copy the connection string from the [Neon dashboard](https://console.neon.tech/)."},
	},
	"rds": {
		DatabaseComment: "# Amazon RDS — use sslmode=require and the instance endpoint",
		SetupNotes:      []string{"Use the RDS endpoint from the AWS console and ensure the security group allows your VM/cluster."},
	},
}

var kafka = map[string]Hints{
	"generic": {},
	"aiven": {
		KafkaComment: "# Aiven Kafka — SASL_SSL / SCRAM-SHA-256 credentials from the service overview",
		SetupNotes:   []string{"Create a Kafka service in the [Aiven console](https://console.aiven.io/)."},
	},
	"upstash-kafka": {
		KafkaComment: "# Upstash Kafka — broker URL and SCRAM credentials from the Upstash console",
		SetupNotes:   []string{"Copy broker, username, and password from the [Upstash Kafka dashboard](https://console.upstash.com/)."},
	},
}

var qstash = map[string]Hints{
	"generic": {},
	"upstash-qstash": {
		QStashComment: "# Upstash QStash — token from console; destination must be your public HTTPS webhook URL",
		SetupNotes:    []string{"Create a QStash token in the [Upstash console](https://console.upstash.com/)."},
	},
}

func DatabaseHints(preset string) Hints {
	return lookup(database, preset)
}

func KafkaHints(preset string) Hints {
	return lookup(kafka, preset)
}

func QStashHints(preset string) Hints {
	return lookup(qstash, preset)
}

func lookup(table map[string]Hints, preset string) Hints {
	if preset == "" {
		preset = "generic"
	}
	if h, ok := table[preset]; ok {
		return h
	}
	return table["generic"]
}

func Normalize(preset string) string {
	if preset == "" {
		return "generic"
	}
	return preset
}

func ValidDatabasePreset(preset string) bool {
	_, ok := database[Normalize(preset)]
	return ok
}

func ValidKafkaPreset(preset string) bool {
	_, ok := kafka[Normalize(preset)]
	return ok
}

func ValidQStashPreset(preset string) bool {
	_, ok := qstash[Normalize(preset)]
	return ok
}

func DatabaseChoices() []string {
	return keys(database)
}

func KafkaChoices() []string {
	return keys(kafka)
}

func QStashChoices() []string {
	return keys(qstash)
}

func keys(m map[string]Hints) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func FormatChoiceList(choices []string) string {
	return fmt.Sprintf("[%s]", join(choices))
}

func join(items []string) string {
	if len(items) == 0 {
		return ""
	}
	s := items[0]
	for i := 1; i < len(items); i++ {
		s += ", " + items[i]
	}
	return s
}
