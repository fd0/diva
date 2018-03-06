# diva

Edit any input field with vim!

# Installation

The program requires Go version 1.7 or newer to compile. To build diva, run the
following command:

```shell
$ go run build.go
```

Afterwards please find a binary of diva in the current directory.

# Integration with i3

In the config file:

```text
bindsym --release $mod+z exec diva
for_window [title="diva-edit-"] "floating enable; resize set 1000 800"
```
