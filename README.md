# Go-Download-Static-Files
Go Download Static Files，使用 Go 下载静态文件（类似 Nginx 搭建静态文件服务）

# 使用
```
默认当前目录，默认端口 8080
Go-Download-Static-Files
Go-Download-Static-Files -port=8080 -root="D:\temp\seata"
Go-Download-Static-Files --port=8080 --root="D:\\temp\\seata"
```
注意事项：  
根目录下最好不要存在"download"、"view"目录，解析会报错。