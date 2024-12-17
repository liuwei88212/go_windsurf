代码功能：
模拟httpproxy，在postmain里面配置代理地址：127.0.0.1:8088

请求示例：
```
curl -x 192.168.81.103:8088 --request GET 'http://www.baidu.com' --header 'Connection: keep-alive'
```

![alt text](image.png)