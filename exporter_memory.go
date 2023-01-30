package main

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterExporter("memory", newExporterMemory)
}

var (
	memoryLabels = []string{"clustername", "node", "self"}

	memoryGaugeVec = map[string]*prometheus.GaugeVec{
		"memory.allocated_unused": newGaugeVec(
			"memory_allocated_unused_bytes",
			"Memory preallocated by the runtime (VM allocators) but not yet used",
			memoryLabels,
		),
		"memory.atom": newGaugeVec(
			"memory_atom_bytes",
			"Memory used by atoms. Should be fairly constant",
			memoryLabels,
		),
		"memory.binary": newGaugeVec(
			"memory_binary_bytes",
			"Memory used by shared binary data in the runtime. Most of this memory is message bodies and metadata.)",
			memoryLabels,
		),
		"memory.code": newGaugeVec(
			"memory_code_bytes",
			"Memory used by code (bytecode, module metadata). This section is usually fairly constant and relatively small (unless the node is entirely blank and stores no data).",
			memoryLabels,
		),
		"memory.connection_channels": newGaugeVec(
			"memory_connection_channels_bytes",
			"Memory used by client connections - channels",
			memoryLabels,
		),
		"memory.connection_other": newGaugeVec(
			"memory_connection_other_bytes",
			"Memory used by client connection - other",
			memoryLabels,
		),
		"memory.connection_readers_bytes": newGaugeVec(
			"memory_connection_readers",
			"Memory used by processes responsible for connection parser and most of connection state. Most of their memory attributes to TCP buffers",
			memoryLabels,
		),
		"memory.connection_writers": newGaugeVec(
			"memory_connection_writers_bytes",
			"Memory used by processes responsible for serialization of outgoing protocol frames and writing to client connections socktes",
			memoryLabels,
		),
		"memory.metrics": newGaugeVec(
			"memory_metrics_bytes",
			"Node-local metrics. The more connections, channels, queues are node hosts, the more stats there are to collect and keep",
			memoryLabels,
		),
		"memory.mgmt_db": newGaugeVec(
			"memory_mgmt_db_bytes",
			"Management DB ETS tables + processes",
			memoryLabels,
		),
		"memory.mnesia": newGaugeVec(
			"memory_mnesia_bytes",
			"Internal database (Mnesia) tables keep an in-memory copy of all its data (even on disc nodes)",
			memoryLabels,
		),
		"memory.msg_index": newGaugeVec(
			"memory_msg_index_bytes",
			"Message index ETS + processes",
			memoryLabels,
		),
		"memory.other_ets": newGaugeVec(
			"memory_other_ets_bytes",
			"Other in-memory tables besides those belonging to the stats database and internal database tables",
			memoryLabels,
		),
		"memory.other_proc": newGaugeVec(
			"memory_other_proc_bytes",
			"Memory used by all other processes that RabbitMQ cannot categorise",
			memoryLabels,
		),
		"memory.other_system": newGaugeVec(
			"memory_other_system_bytes",
			"Memory used by all other system that RabbitMQ cannot categorise",
			memoryLabels,
		),
		"memory.plugins": newGaugeVec(
			"memory_plugins_bytes",
			"Memory used by plugins (apart from the Erlang client which is counted under Connections, and the management database which is counted separately). ",
			memoryLabels,
		),
		"memory.queue_procs": newGaugeVec(
			"memory_queue_procs_bytes",
			"Memory used by class queue masters, queue indices, queue state",
			memoryLabels,
		),
		"memory.queue_slave_procs": newGaugeVec(
			"memory_queue_slave_procs_bytes",
			"Memory used by class queue mirrors, queue indices, queue state",
			memoryLabels,
		),
		"memory.reserved_unallocated": newGaugeVec(
			"memory_reserved_unallocated_bytes",
			"Memory preallocated/reserved by the kernel but not the runtime",
			memoryLabels,
		),
		"memory.total.allocated": newGaugeVec(
			"memory_total_allocated_bytes",
			"Node-local total memory - allocated",
			memoryLabels,
		),
		"memory.total.rss": newGaugeVec(
			"memory_total_rss_bytes",
			"Node-local total memory - rss",
			memoryLabels,
		),
		"memory.total.erlang": newGaugeVec(
			"memory_total_erlang_bytes",
			"Node-local total memory - erlang",
			memoryLabels,
		),
	}
)

type exporterMemory struct {
	memoryMetricsGauge map[string]*prometheus.GaugeVec
}

func newExporterMemory() Exporter {
	memoryGaugeVecActual := memoryGaugeVec

	if len(config.ExcludeMetrics) > 0 {
		for _, metric := range config.ExcludeMetrics {
			if memoryGaugeVecActual[metric] != nil {
				delete(memoryGaugeVecActual, metric)
			}
		}
	}

	return exporterMemory{
		memoryMetricsGauge: memoryGaugeVecActual,
	}
}

func (e exporterMemory) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	for _, gauge := range e.memoryMetricsGauge {
		gauge.Reset()
	}
	selfNode := ""
	if n, ok := ctx.Value(nodeName).(string); ok {
		selfNode = n
	}
	cluster := ""
	if n, ok := ctx.Value(clusterName).(string); ok {
		cluster = n
	}

	nodeData, err := getStatsInfo(config, "nodes", nodeLabelKeys)
	if err != nil {
		return err
	}

	for _, node := range nodeData {
		self := selfLabel(config, node.labels["name"] == selfNode)
		rabbitMemoryResponses, err := getMetricMap(config, fmt.Sprintf("nodes/%s/memory", node.labels["name"]))
		if err != nil {
			return err
		}
		for key, gauge := range e.memoryMetricsGauge {
			if value, ok := rabbitMemoryResponses[key]; ok {
				gauge.WithLabelValues(cluster, node.labels["name"], self).Set(value)
			}
		}
	}

	for _, gauge := range e.memoryMetricsGauge {
		gauge.Collect(ch)
	}
	return nil
}

func (e exporterMemory) Describe(ch chan<- *prometheus.Desc) {
	for _, gauge := range e.memoryMetricsGauge {
		gauge.Describe(ch)
	}
}
