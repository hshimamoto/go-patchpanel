all: patchpanel patchlink

patchpanel::
	cd patchpanel; go get; go build

patchlink::
	cd patchlink; go get; go build

clean:
	rm -f patchpanel/patchpanel
	rm -f patchlink/patchlink
