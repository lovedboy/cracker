# cracker
proxy over http[s], support http,socks5 proxy.

```
+------------+            +--------------+          
| local app  |  <=======> |local proxy   | <#######
+------------+            +--------------+        #
                                                  #
                                                  #
                                                  # http[s]
                                                  #
                                                  #
+-------------+            +--------------+       #
| target host |  <=======> |http[s] server|  <#####
+-------------+            +--------------+         
```

# Install

Download the latest binaries from this [release page](https://github.com/lovedboy/cracker/releases).

You can also install from source if you have go installed.

```
git clone https://github.com/lovedboy/cracker
cd cracker
make install
cd bin
list
```
# Usage

## Server side (Run on your vps or other application container platform)

```
./server -addr :8080 -secret <password>
```

## Local side (Run on your local pc)

```
./local -raddr http://example.com:8080 -secret <password>
```

## https

It is strongly recommended to open the https option on the server side.

### Notice

The file name of certificate and private key must be `cert.pem` and `key.pem` and with the server bin under the same folder.

If you have a ssl certificate, It would be easy.

copy the certificate and private key into the same folder with server bin

```
./server -addr :8080 -secret <password> -https
```

```
./local -raddr https://example.com:8080 -secret <password>
```

Of Course, you can create a self-signed ssl certificate by openssl.

```
openssl req -subj '/CN=*/' -x509 -sha256 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 1024 -nodes
```

```
./server -addr :8080 -secret <password> -https
```
copy the certificate into the same folder with local bin and bind the ip and hostname(not domain !!!)

```
echo "<your server ip> <hostname>" >> /etc/hosts
./local -raddr https://<hostname>:8080 -secret <password>
```

## Next

Play with [SwitchyOmega](https://github.com/FelisCatus/SwitchyOmega/releases)


