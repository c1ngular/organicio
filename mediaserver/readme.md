执行可执行程序时，请在终端输入：

```
export DYLD_LIBRARY_PATH=./
./MediaServer -d &
```

如果由于so动态库链接失败导致运行不起来，请重建so库软链接
如果由于端口权限问题导致启动失败，请修改配置文件中端口号，或者以root权限运行
