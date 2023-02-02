# 用go使用rocksdb
rocksdb是Facebook用C写的，有些项目封装了一层外皮用go去调C，然而rocksdb的版本一直在升级，有些C函数在新版本中已经不存在了，go调不到就报错，所以这里面最麻烦的事情就是把版本对应上。    
## 1.准备工作
升级gcc的版本至少到4.8以支持C++11的特性。  
```shell
sudo apt-get -y install build-essential
sudo apt-get -y install libgflags-dev libsnappy-dev zlib1g-dev libbz2-dev liblz4-dev libzstd-dev
```
## 2.安装rocksdb
我打算把rocksdb安装到`/home/ubuntu/software`目录下。  
```shell
cd /home/ubuntu/software
git clone https://github.com/facebook/rocksdb.git
cd rocksdb
git checkout v6.17.3  
make shared_lib
```
注意我没有使用最新版本的rocksdb源码，而是切回到了v6.17.3这个tag，因为版本太新的话line/gorocksdb/options.go里用到的一些C函数在github.com/facebook/rocksdb/include/rocksdb/c.h里找不到；版本太老的话有些语法用新版的gcc已经编译不通过了。  
make shared_lib会编译生成动态库librocksdb.so，接下来要用。  
还需要设置几个环境变量：  
```shell
vim /etc/profile
export CPLUS_INCLUDE_PATH=${CPLUS_INCLUDE_PATH}:/home/ubuntu/software/rocksdb/include
export LD_LIBRARY_PATH=${LD_LIBRARY_PATH}:/home/ubuntu/software/rocksdb
export LIBRARY_PATH=${LIBRARY_PATH}:/home/ubuntu/software/rocksdb
source /etc/profile
```
## 3.安装tecbot/gorocksdb
```shell
cd /path/to/your/go_project
go get github.com/line/gorocksdb@v0.0.0-20220620070102-28c3d9c31088
CGO_CFLAGS="-I/home/ubuntu/software/rocksdb/include"  CGO_LDFLAGS="-L/home/ubuntu/software/rocksdb -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd" go build -o my_exe main.go
./my_exe
```  
执行go get必须在某个包含go.mod的目录下。  
执行go get时需要的几个动态库-lrocksdb在第2步时生成，其他几个-lsnappy -llz4 -lzstd等在第1步时生成。  
执行my_exe之前需要把LD_LIBRARY_PATH等环境变量设置好，不然找不到动态库，这是在第2步做的。  
line/gorocksdb项目是从tecbot/gorocksdb项目fork过来的，之所以没用tecbot/gorocksdb是因为它太久没更新了，只能跟rocksdb v5.x对应上，而rocksdb v5.x任何一个版本的代码用新版的gcc已经编译不通过了（我试了几个v5版本都不能正常编译）。这里的line/gorocksdb我故意指定了版本v0.0.0-20220620070102-28c3d9c31088，就是担心将来的版本又跟rocksdb v6.17.3对应不上。     