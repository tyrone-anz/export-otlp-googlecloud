package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	sdkmetric "go.opentelemetry.io/otel/sdk/export/metric"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

func main() {
	ctx := context.Background()
	host := "localhost:55680"

	client := otlpmetricgrpc.NewClient(
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(host),
	)

	exporter, err := otlpmetric.New(ctx, client, otlpmetric.WithMetricExportKindSelector(sdkmetric.DeltaExportKindSelector()))
	if err != nil {
		fmt.Println("error %v", err)
		os.Exit(1)
	}

	// cont := controller.New(processor.New(selector.NewWithInexpensiveDistribution(), exporter),
	cont := controller.New(processor.New(selector.NewWithExactDistribution(), exporter),
		controller.WithExporter(exporter),
		controller.WithCollectPeriod(time.Second*2))

	if err := cont.Start(ctx); err != nil {
		fmt.Println("error %v", err)
		os.Exit(1)
	}

	global.SetMeterProvider(cont.MeterProvider())

	meter := global.Meter("")

	valuerecorder := metric.Must(meter).NewInt64ValueRecorder("test.dummy.one")

	valuerecorder.Record(ctx, 100, attribute.Any("rpc.method", "Hello"))
	valuerecorder.Record(ctx, 20, attribute.Any("rpc.method", "Hi"))

	time.Sleep(time.Second * 5) // wait for metrics to be collected
}

// Collector config
//
// receivers:
//  otlp:
//    protocols:
//      grpc:
//        endpoint: 0.0.0.0:55680
//  googlecloud:
//    # Options are defined here: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/master/exporter/stackdriverexporter
//    project: <removed>
//    timeout: 10s
//    metric:
//      prefix: <removed>
//      skip_create_descriptor: false
//  logging:
//    logLevel: debug
// extensions:
//  health_check:
//    port: 55666 # Path /
//  pprof:
//    endpoint: 0.0.0.0:55667 # Path /debug/pprof/
//  zpages:
//    endpoint: 0.0.0.0:55668 # Path /debug/rpcz /debug/tracez
// service:
//  extensions: [health_check,pprof,zpages]
//  pipelines:
//    metrics:
//      receivers: [otlp]
//      processors: [memory_limiter, batch]
//      exporters: [googlecloud, logging]

// When using inexpensive distribution as aggregator selector
//
// Logs from collector:
//
// 2021-08-26T13:33:09.837+1000    DEBUG   loggingexporter/logging_exporter.go:66  ResourceMetrics #0
// Resource labels:
//     -> service.name: STRING(unknown_service:___go_build_main_go)
//     -> telemetry.sdk.language: STRING(go)
//     -> telemetry.sdk.name: STRING(opentelemetry)
//     -> telemetry.sdk.version: STRING(1.0.0-RC1)
// InstrumentationLibraryMetrics #0
// InstrumentationLibrary
// Metric #0
// Descriptor:
//     -> Name: test.dummy.one
//     -> Description:
//     -> Unit:
//     -> DataType: Summary
// SummaryDataPoints #0
// StartTimestamp: 2021-08-26 03:33:07.74398 +0000 UTC
// Timestamp: 2021-08-26 03:33:09.746479 +0000 UTC
// Count: 1
// Sum: 100.000000
// QuantileValue #0: Quantile 0.000000, Value 100.000000
// QuantileValue #1: Quantile 1.000000, Value 100.000000
// SummaryDataPoints #1
// StartTimestamp: 2021-08-26 03:33:07.74398 +0000 UTC
// Timestamp: 2021-08-26 03:33:09.746479 +0000 UTC
// Count: 1
// Sum: 20.000000
// QuantileValue #0: Quantile 0.000000, Value 20.000000
// QuantileValue #1: Quantile 1.000000, Value 20.000000
//
// 2021-08-26T13:33:14.505+1000    info    exporterhelper/queued_retry.go:325      Exporting failed. Will retry the request after interval.
//   {"kind": "exporter", "name": "googlecloud", "error": "rpc error: code = InvalidArgument desc = One or more TimeSeries could not be written: Field timeSeries[4] had an invalid value: Duplicate TimeSeries encountered. Only one point can be written per TimeSeries per request.: timeSeries[4]; Field timeSeries[5] had an invalid value: Duplicate TimeSeries encountered. Only one point can be written per TimeSeries per request.: timeSeries[5]; Field timeSeries[6] had an invalid value: Duplicate TimeSeries encountered. Only one point can be written per TimeSeries per request.: timeSeries[6]; Field timeSeries[7] had an invalid value: Duplicate TimeSeries encountered. Only one point can be written per TimeSeries per request.: timeSeries[7]", "interval": "4.397092313s"}
//
//
// Logged JSON of the proto sent to collector:
//
// {
// "resource_metrics": [
//  {
//   "resource": {
//    "attributes": [
//     {
//      "key": "service.name",
//      "value": {
//       "Value": {
//        "StringValue": "unknown_service:___go_build_main_go"
//       }
//      }
//     },
//     {
//      "key": "telemetry.sdk.language",
//      "value": {
//       "Value": {
//        "StringValue": "go"
//       }
//      }
//     },
//     {
//      "key": "telemetry.sdk.name",
//      "value": {
//       "Value": {
//        "StringValue": "opentelemetry"
//       }
//      }
//     },
//     {
//      "key": "telemetry.sdk.version",
//      "value": {
//       "Value": {
//        "StringValue": "1.0.0-RC1"
//       }
//      }
//     }
//    ]
//   },
//   "instrumentation_library_metrics": [
//    {
//     "metrics": [
//      {
//       "name": "test.dummy.one",
//       "Data": {
//        "Summary": {
//         "data_points": [
//          {
//           "attributes": [
//            {
//             "key": "rpc.method",
//             "value": {
//              "Value": {
//               "StringValue": "Hello"
//              }
//             }
//            }
//           ],
//           "start_time_unix_nano": 1629948787743980000,
//           "time_unix_nano": 1629948789746479000,
//           "count": 1,
//           "sum": 100,
//           "quantile_values": [
//            {
//             "value": 100
//            },
//            {
//             "quantile": 1,
//             "value": 100
//            }
//           ]
//          },
//          {
//           "attributes": [
//            {
//             "key": "rpc.method",
//             "value": {
//              "Value": {
//               "StringValue": "Hi"
//              }
//             }
//            }
//           ],
//           "start_time_unix_nano": 1629948787743980000,
//           "time_unix_nano": 1629948789746479000,
//           "count": 1,
//           "sum": 20,
//           "quantile_values": [
//            {
//             "value": 20
//            },
//            {
//             "quantile": 1,
//             "value": 20
//            }
//           ]
//          }
//         ]
//        }
//       }
//      }
//     ]
//    }
//   ]
//  }
// ]
// }

// When using exact distribution as aggregator selector
//
// Logs from collector:
//
// 2021-08-26T13:20:57.984+1000    DEBUG   loggingexporter/logging_exporter.go:66  ResourceMetrics #0
// Resource labels:
//     -> service.name: STRING(unknown_service:___go_build_main_go)
//     -> telemetry.sdk.language: STRING(go)
//     -> telemetry.sdk.name: STRING(opentelemetry)
//     -> telemetry.sdk.version: STRING(1.0.0-RC1)
// InstrumentationLibraryMetrics #0
// InstrumentationLibrary
// Metric #0
// Descriptor:
//     -> Name: test.dummy.one
//     -> Description:
//     -> Unit:
//     -> DataType: Gauge
// NumberDataPoints #0
// StartTimestamp: 2021-08-26 03:20:55.876357 +0000 UTC
// Timestamp: 2021-08-26 03:20:57.878361 +0000 UTC
// Value: 100
// NumberDataPoints #1
// StartTimestamp: 2021-08-26 03:20:55.876357 +0000 UTC
// Timestamp: 2021-08-26 03:20:57.878361 +0000 UTC
// Value: 20
//
// 2021-08-26T13:21:02.635+1000    info    exporterhelper/queued_retry.go:325      Exporting failed. Will retry the request after interval.
//   {"kind": "exporter", "name": "googlecloud", "error": "rpc error: code = Internal desc = One or more TimeSeries could not be written: Field timeSeries[1] had an invalid value: Duplicate TimeSeries encountered. Only one point can be written per TimeSeries per request.: timeSeries[1]; Internal error encountered. Please retry after a few seconds. If internal errors persist, contact support at https://cloud.google.com/support/docs.: timeSeries[0]", "interval": "7.461358278s"}
//
//
// Logged JSON of the proto sent to collector:
//
// {
// "resource_metrics": [
//  {
//   "resource": {
//    "attributes": [
//     {
//      "key": "service.name",
//      "value": {
//       "Value": {
//        "StringValue": "unknown_service:___go_build_main_go"
//       }
//      }
//     },
//     {
//      "key": "telemetry.sdk.language",
//      "value": {
//       "Value": {
//        "StringValue": "go"
//       }
//      }
//     },
//     {
//      "key": "telemetry.sdk.name",
//      "value": {
//       "Value": {
//        "StringValue": "opentelemetry"
//       }
//      }
//     },
//     {
//      "key": "telemetry.sdk.version",
//      "value": {
//       "Value": {
//        "StringValue": "1.0.0-RC1"
//       }
//      }
//     }
//    ]
//   },
//   "instrumentation_library_metrics": [
//    {
//     "metrics": [
//      {
//       "name": "test.dummy.one",
//       "Data": {
//        "Gauge": {
//         "data_points": [
//          {
//           "attributes": [
//            {
//             "key": "rpc.method",
//             "value": {
//              "Value": {
//               "StringValue": "Hello"
//              }
//             }
//            }
//           ],
//           "start_time_unix_nano": 1629948324284575000,
//           "time_unix_nano": 1629948326288973000,
//           "Value": {
//            "AsInt": 100
//           }
//          },
//          {
//           "attributes": [
//            {
//             "key": "rpc.method",
//             "value": {
//              "Value": {
//               "StringValue": "Hi"
//              }
//             }
//            }
//           ],
//           "start_time_unix_nano": 1629948324284575000,
//           "time_unix_nano": 1629948326288973000,
//           "Value": {
//            "AsInt": 20
//           }
//          }
//         ]
//        }
//       }
//      }
//     ]
//    }
//   ]
//  }
// ]
// }
