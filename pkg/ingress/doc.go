// Package ingress parses landscape documents (YAML, JSON, CSV) and applies them to [model.Model].
//
// Sensors obtain bytes from external origins and call [Parse] with [ParseOptions];
// format parsing and model apply live here so any file-shaped Sensor can reuse them.
package ingress
