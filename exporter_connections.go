package main

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterExporter("connections", newExporterConnections)
}

var (
	connectionLabels            = []string{"clustername", "vhost", "node", "peer_host", "user", "self"}
	connectionLabelsStateMetric = []string{"clustername", "vhost", "node", "peer_host", "user", "state", "self"}
	connectionLabelKeys         = []string{"vhost", "node", "peer_host", "user", "state", "node"}

	connectionGaugeVec = map[string]*prometheus.GaugeVec{
		"channels":  newGaugeVec("connection_channels", "number of channels in use", connectionLabels),
		"recv_oct":  newGaugeVec("connection_received_bytes", "received bytes", connectionLabels),
		"recv_cnt":  newGaugeVec("connection_received_packets", "received packets", connectionLabels),
		"send_oct":  newGaugeVec("connection_send_bytes", "send bytes", connectionLabels),
		"send_cnt":  newGaugeVec("connection_send_packets", "send packets", connectionLabels),
		"send_pend": newGaugeVec("connection_send_pending", "Send queue size", connectionLabels),
	}
)

type exporterConnections struct {
	connectionMetricsG map[string]*prometheus.GaugeVec
	stateMetric        *prometheus.GaugeVec
}

func newExporterConnections() Exporter {
	connectionGaugeVecActual := connectionGaugeVec

	if len(config.ExcludeMetrics) > 0 {
		for _, metric := range config.ExcludeMetrics {
			if connectionGaugeVecActual[metric] != nil {
				delete(connectionGaugeVecActual, metric)
			}
		}
	}

	return exporterConnections{
		connectionMetricsG: connectionGaugeVecActual,
		stateMetric:        newGaugeVec("connection_status", "Number of connections in a certain state aggregated per label combination.", connectionLabelsStateMetric),
	}
}

func (e exporterConnections) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	rabbitConnectionResponses, err := getStatsInfo(config, "connections", connectionLabelKeys)

	if err != nil {
		return err
	}
	for _, gauge := range e.connectionMetricsG {
		gauge.Reset()
	}
	e.stateMetric.Reset()

	selfNode := ""
	if n, ok := ctx.Value(nodeName).(string); ok {
		selfNode = n
	}
	cluster := ""
	if n, ok := ctx.Value(clusterName).(string); ok {
		cluster = n
	}

	for key, gauge := range e.connectionMetricsG {
		for _, connD := range rabbitConnectionResponses {
			if value, ok := connD.metrics[key]; ok {
				self := selfLabel(config, connD.labels["node"] == selfNode)
				gauge.WithLabelValues(cluster, connD.labels["vhost"], connD.labels["node"], connD.labels["peer_host"], connD.labels["user"], self).Add(value)
			}
		}
	}

	for _, connD := range rabbitConnectionResponses {
		self := selfLabel(config, connD.labels["node"] == selfNode)
		e.stateMetric.WithLabelValues(cluster, connD.labels["vhost"], connD.labels["node"], connD.labels["peer_host"], connD.labels["user"], connD.labels["state"], self).Add(1)
	}

	for _, gauge := range e.connectionMetricsG {
		gauge.Collect(ch)
	}
	e.stateMetric.Collect(ch)
	return nil
}

func (e exporterConnections) Describe(ch chan<- *prometheus.Desc) {
	for _, nodeMetric := range e.connectionMetricsG {
		nodeMetric.Describe(ch)
	}

}
