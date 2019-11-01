### 将语雀指定知识库的文章同步到Hugo中

在hugo的配置文件中加入下面的配置，然后下载yuque_sync到hugo的源代码目录，在hugo命令之前执行yuque_sync就可以将文章同步过去，
点击进入知识库后查看类似这样的URL https://www.yuque.com/[user]/[kb]
其中`user`和`kb`分别对应下面配置的对应项，`token`是如果知识库是私密的就需要token，否则可以留空白
```toml
[yuque-sync]
# 用户名
user = ""
# 知识库路径
kb = ""
# 私密仓库需要token
token = ""
# api
api = "https://www.yuque.com/api/v2"
# port
port = 8081
# sync after command
aftercmd = ""
```
请在`hugo博客主目录`运行,同步后的文章会存储在`content/post`,如果你在文章中使用了语雀来插入图片，那么图片会被下载到本地的`content/images/`目录下,并且同步的文档里的图片链接会被替换为`/image/imagepath.png`,请将`hugo`命令加入环境变量中
web
语雀还可以设置webhook，所以可以结合travis来触发自动拉取最新的文章后自动构建博客