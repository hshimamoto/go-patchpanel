all: patchpanel patchlink

patchpanel::
	cd patchpanel; go build

patchlink::
	cd patchlink; go build

clean:
	rm -f patchpanel/patchpanel
	rm -f patchlink/patchlink
