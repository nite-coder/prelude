./setup.sh 10000 5 127.0.0.1  (不要用 host.docker.internal 會很慢)

./destory.sh

可以用這個來查線上人數，但不要用 browser 開，因為 browser 不能打開 port 10080
<http://localhost:10080/status>

socket抛出Can't assign requested address
sysctl -w net.ipv4.tcp_tw_reuse=1 && sysctl -w net.ipv4.tcp_fin_timeout=10 && sysctl -w net.ipv4.ip_local_port_range='1024 65535'
