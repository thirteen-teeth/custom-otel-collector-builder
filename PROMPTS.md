```
I would like to develop a GELF input plugin for the opentelemtry collector and create custom otel-collector docker image that supports my new input

using this repo as the place to build from, can you follow the documentation on this URL https://opentelemetry.io/docs/collector/building/receiver/

and the GELF specification on this URL https://go2docs.graylog.org/current/getting_in_log_data/gelf.html

please create a detailed plan for how to perform those actions and ask any questions required to implement

only support tcp & udp, not http
default to use GELF port 12201 but have the listening port and protocol be configurable
please only support gzip
log errors
please make best effort to be as performant as possible, should support existing opentelemetry collector processesors such as batch
no custom transformations at this moment
```