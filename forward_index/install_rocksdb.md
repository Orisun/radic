# 用go使用rocksdb
rocksdb是Facebook用C写的，有些项目封装了一层外皮用go去调C，然而rocksdb的版本一直在升级，有些C函数在新版本中已经不存在了，go调不到就报错，所以这里面最麻烦的事情就是把版本对应上。    
## 准备工作
升级gcc的版本至少到4.8以支持C++11的特性。  
```shell
apt-get -y install build-essential
apt-get -y install libgflags-dev libsnappy-dev zlib1g-dev libbz2-dev liblz4-dev libzstd-dev
```
## 安装rocksdb
我打算把rocksdb安装到`/home/ubuntu/software`目录下。  
```shell
cd /home/ubuntu/software
git clone https://github.com/facebook/rocksdb.git
cd rocksdb
git checkout v6.17.3  
make shared_lib
```
注意我没有使用最新版本的rocksdb源码，而是切回到了v6.17.3 tag，因为版本太新的话line/gorocksdb/options.go里用到的一些C函数在github.com/facebook/rocksdb/include/rocksdb/c.h里找不到；版本太老的话有些语法用新版的gcc已经编译不通过了。  
make shared_lib会编译生成动态库librocksdb.so，接下来要用。  
## 安装tecbot/gorocksdb
```shell
cd /path/to/your/go_project
go get github.com/line/gorocksdb@v0.0.0-20220620070102-28c3d9c31088
CGO_CFLAGS="-I/home/ubuntu/software/rocksdb/include"  CGO_LDFLAGS="-L/home/ubuntu/software/rocksdb -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd" go build main.go
```
line/gorocksdb项目是从tecbot/gorocksdb项目fork过来的，之所以没用tecbot/gorocksdb是因为它太久没更新了，只能跟rocksdb v5.x对应上，