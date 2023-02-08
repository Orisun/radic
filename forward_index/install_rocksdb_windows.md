# 用go使用rocksdb（windows版）
rocksdb是Facebook用C写的，有些项目封装了一层外皮用go去调C，然而rocksdb的版本一直在升级，有些C函数在新版本中已经不存在了，go调不到就报错，所以这里面最麻烦的事情就是把版本对应上。    
## 1.准备工作
1. 64位的windows安装mingw x86_64-posix-silj版本(silj同时支持64和32位，更稳定)，然后配置环境变量，mingw64\bin放到Path中，mingw64\lib放到Lib中，mingw64\include放到INCLUDE中。配置环境变量CC=D:\mingw64\bin\gcc.exe，否则安装rocksdb时会报找不到编译器。    
2. 安装Visual Studio。打开Visual Studio安装向导，修改安装，点语言包，勾选英语，安装英语包，否则后续安装rocksdb时会报错：The following VS instances are excluded because the English language pack is unavailable。 
## 2.安装vcpkg
我打算把vcpkg安装到D:\software，所以`cd d:\software`
```shell
git clone git@github.com:microsoft/vcpkg.git
cd vcpkg
.\bootstrap-vcpkg.bat 
```
最后一步会生成vcpkg.exe文件，将来就用它来安装rocksdb。把d:\software\vcpkg添加到环境变量的Path里，这样就可以任意目录下使用vcpkg命令了。  
## 3.安装rocksdb
在linux上我用的rocksdb版本是6.17.3，在windows上vcpkg不支持安装这个版本，跟它比较接近的版本是6.26.1，所以打算通过vcpkg安装这个版本的rocksdb。  
随便建一个目录，比如叫project6-26-1，在project6-26-1下新建一个文件vcpkg.json，内容如下：  
```json
{
  "name": "my-project",
  "version": "1.0",
  "dependencies": [
     { 
       "name": "rocksdb", 
       "version>=": "7.7.3" 
     }
  ],
  "builtin-baseline":"185a7aa23ac037d711a2a87d5e8d99e7131dbcd8",
  "overrides": [
    { "name": "rocksdb", "version": "6.26.1" }
  ]
}
```
头两行是项目的名称和版本号，随便写就行，注意项目名里不能包含下划线。dependencies就是要编译安装的库，这里指定了rocksdb的最低版本是7.7.3，跟我们需要的6.26.1是矛盾的，没关系，在下面的overrides里又明确指定了rocksdb的版本是6.26.1。至于builtin-baseline写你git clone的vcpkg的commit id就行。  
来到project6-26-1目录下执行
```Shell
vcpkg install --triplet=x64-windows
```
vcpkg默认是安装32位的库，在本机需要安装x64的库，因此需要添加一个系统变量重新指定默认的TRIPLET：VCPKG_DEFAULT_TRIPLET=x64-windows或者给install传入一个选项--triplet=x64-windows。这个x64-windows默认是编译成x64的动态链接库dll，如果需要静态链接库lib，选择x64-windows-static。  
## 3.安装line/gorocksdb
```shell
cd /path/to/your/go_project
go get github.com/line/gorocksdb@v0.0.0-20220620070102-28c3d9c31088
```
由于跟line/gorocksdb@v0.0.0-20220620070102-28c3d9c31088兼容的rocksdb版本是6.17.3，而我们装的是 rocksdbV6.26.1，需要修改源文件$GOPATH/pkg/mod/github.com/line/gorocksdb@v0.0.0-20220620070102-28c3d9c31088/filter_policy.go。  
把第49行的
```C.rocksdb_filterpolicy_create_bloom(C.int(bitsPerKey))```
改为
```C.rocksdb_filterpolicy_create_bloom(C.double(bitsPerKey))```
把第55行的
```C.rocksdb_filterpolicy_create_bloom_full(C.int(bitsPerKey))```
改为
```C.rocksdb_filterpolicy_create_bloom_full(C.double(bitsPerKey))```
设置go env
```Shell
go env -w CGO_CFLAGS="-ID:\software\vcpkg_installed\project6-26-1\vcpkg_installed\x64-windows\include"  CGO_LDFLAGS="-LD:\software\vcpkg_installed\project6-26-1\vcpkg_installed\x64-windows\lib -lrocksdb -lstdc++ -lm -ldl"
```
编译go项目
```Shell
go build -o my_exe main.go
```