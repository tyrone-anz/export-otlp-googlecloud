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

// This file tests the exporting of metrics (value recorder kind) to collector then collector to google cloud.
// There are two recorded data for the metric with different attribute value.
// Regardless of the selector aggregator used, google cloud exporter on the collector throws the `Duplicate Timeseries` error.
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
		// cont := controller.New(processor.New(selector.NewWithHistogramDistribution(), exporter),
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
	valuerecorder.Record(ctx, 20, attribute.Any("rpc.method", "Hi"))
	valuerecorder.Record(ctx, 25, attribute.Any("rpc.method", "Hi"))
	valuerecorder.Record(ctx, 25, attribute.Any("rpc.method", "Hi"))

	time.Sleep(time.Second * 5) // wait for metrics to be collected
}

// Collector config (v0.31.0)
// https://github.com/open-telemetry/opentelemetry-collector-contrib
//
// receivers:
//  otlp:
//    protocols:
//      grpc:
//        endpoint: 0.0.0.0:55680
//  googlecloud:
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

// Case #1: When using inexpensive distribution as aggregator selector ================================================================
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

// Case #2: When using exact distribution as aggregator selector ================================================================
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

// Case #3: When using histogram distribution as aggregator selector ================================================================
//
// Logs from collector:
//
// 2021-08-26T13:45:35.744+1000    DEBUG   loggingexporter/logging_exporter.go:66  ResourceMetrics #0
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
//     -> DataType: Histogram
//     -> AggregationTemporality: AGGREGATION_TEMPORALITY_DELTA
// HistogramDataPoints #0
// StartTimestamp: 2021-08-26 03:45:33.621217 +0000 UTC
// Timestamp: 2021-08-26 03:45:35.621554 +0000 UTC
// Count: 1
// Sum: 100.000000
// ExplicitBounds #0: 5000.000000
// ExplicitBounds #1: 10000.000000
// ExplicitBounds #2: 25000.000000
// ExplicitBounds #3: 50000.000000
// ExplicitBounds #4: 100000.000000
// ExplicitBounds #5: 250000.000000
// ExplicitBounds #6: 500000.000000
// ExplicitBounds #7: 1000000.000000
// ExplicitBounds #8: 2500000.000000
// ExplicitBounds #9: 5000000.000000
// ExplicitBounds #10: 10000000.000000
// Buckets #0, Count: 1
// Buckets #1, Count: 0
// Buckets #2, Count: 0
// Buckets #3, Count: 0
// Buckets #4, Count: 0
// Buckets #5, Count: 0
// Buckets #6, Count: 0
// Buckets #7, Count: 0
// Buckets #8, Count: 0
// Buckets #9, Count: 0
// Buckets #10, Count: 0
// Buckets #11, Count: 0
// HistogramDataPoints #1
// StartTimestamp: 2021-08-26 03:45:33.621217 +0000 UTC
// Timestamp: 2021-08-26 03:45:35.621554 +0000 UTC
// Count: 1
// Sum: 20.000000
// ExplicitBounds #0: 5000.000000
// ExplicitBounds #1: 10000.000000
// ExplicitBounds #2: 25000.000000
// ExplicitBounds #3: 50000.000000
// ExplicitBounds #4: 100000.000000
// ExplicitBounds #5: 250000.000000
// ExplicitBounds #6: 500000.000000
// ExplicitBounds #7: 1000000.000000
// ExplicitBounds #8: 2500000.000000
// ExplicitBounds #9: 5000000.000000
// ExplicitBounds #10: 10000000.000000
// Buckets #0, Count: 1
// Buckets #1, Count: 0
// Buckets #2, Count: 0
// Buckets #3, Count: 0
// Buckets #4, Count: 0
// Buckets #5, Count: 0
// Buckets #6, Count: 0
// Buckets #7, Count: 0
// Buckets #8, Count: 0
// Buckets #9, Count: 0
// Buckets #10, Count: 0
// Buckets #11, Count: 0
//
// 2021-08-26T13:45:39.887+1000    info    exporterhelper/queued_retry.go:325      Exporting failed. Will retry the request after interval.
//   {"kind": "exporter", "name": "googlecloud", "error": "rpc error: code = Internal desc = One or more TimeSeries could not be written: Field timeSeries[1] had an invalid value: Duplicate TimeSeries encountered. Only one point can be written per TimeSeries per request.: timeSeries[1]; Internal error encountered. Please retry after a few seconds. If internal errors persist, contact support at https://cloud.google.com/support/docs.: timeSeries[0]", "interval": "3.770024677s"}
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
//       "name": "test.dummy.two",
//       "Data": {
//        "Histogram": {
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
//           "start_time_unix_nano": 1629949533621217000,
//           "time_unix_nano": 1629949535621554000,
//           "count": 1,
//           "sum": 100,
//           "bucket_counts": [
//            1,
//            0,
//            0,
//            0,
//            0,
//            0,
//            0,
//            0,
//            0,
//            0,
//            0,
//            0
//           ],
//           "explicit_bounds": [
//            5000,
//            10000,
//            25000,
//            50000,
//            100000,
//            250000,
//            500000,
//            1000000,
//            2500000,
//            5000000,
//            10000000
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
//           "start_time_unix_nano": 1629949533621217000,
//           "time_unix_nano": 1629949535621554000,
//           "count": 1,
//           "sum": 20,
//           "bucket_counts": [
//            1,
//            0,
//            0,
//            0,
//            0,
//            0,
//            0,
//            0,
//            0,
//            0,
//            0,
//            0
//           ],
//           "explicit_bounds": [
//            5000,
//            10000,
//            25000,
//            50000,
//            100000,
//            250000,
//            500000,
//            1000000,
//            2500000,
//            5000000,
//            10000000
//           ]
//          }
//         ],
//         "aggregation_temporality": 1
//        }
//       }
//      }
//     ]
//    }
//   ]
//  }
// ]
// }
