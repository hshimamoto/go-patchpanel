Patch Panel
===========

Description
-----------
Patch Panel provides SSH connection relay between
`panel` host and `link` host.

```
+------------+    +-------+    +------+    +------+
| SSH client |    |       |    |      |    |      |
|    with    |====| panel |====| link |====| SSHd |
| HTTP Proxy |    |       |    |      |    |      |
+------------+    +-------+    +------+    +------+
```

Front side (Left side)
----------------------
HTTP CONNECT
```
CONNECT name:port HTTP/1.0

```

Back side (Right side)
----------------------
link to panel
`LINK name CRLF`

panel to link
`NEW name CRLF`

link to panel with new connection
`CONNECTED name CRLF`

License
-------
MIT License Copyright (c) 2020 Hiroshi Shimamoto
