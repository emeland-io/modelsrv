// Package filesensor watches Sources for landscape documents and applies them to [model.Model].
//
// Format parsing lives in [go.emeland.io/modelsrv/pkg/ingress]; this package owns
// how bytes are obtained (local directory, HTTP, S3) and Sensor-side parser configuration.
package filesensor
