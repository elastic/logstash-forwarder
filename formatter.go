package main

import (
   "github.com/gemsi/grok"
   "encoding/json"
)

func Formatter(input chan []*FileEvent, output chan []*FileEvent, config *Config) {
    var patterns map[string]string
    patterns = make(map[string]string)

    for _, file := range config.Files {
        for _, path := range file.Paths {
            if file.Format != "" {
                patterns[path] = file.Format
            }
        }
    }

    var g *grok.Grok = grok.NewWithConfig(&grok.Config{NamedCapturesOnly: true})
    var pattern string

	for {
		for events := range input {
			for _, event := range events {
				// skip stdin
				if *event.Source == "-" {
					continue
				}

                pattern = patterns[*event.Source]

                values, _ := g.Parse(pattern, *event.Text)
		        delete(values, "")

		        encoded, _ := json.Marshal(values)
				*event.Text = string(encoded)

			}

			output <- events
		}
	}
}
