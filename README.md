# ksrp

## build image

```
go install github.com/ko-build/ko@latest
export KOCACHE=$HOME/.cache/ko
export CI_REGISTRY=<your registry>
./build_image.sh ./cmd/ksrp-expose
./build_image.sh ./cmd/ksrp-agent
```

## generate kubernetes yaml

```
jetter -v kubernetes/jet-values.yaml kubernetes templates.jet > kubernetes/deploy.yaml
```

## todo

- [x] agent backend 连接池
- [ ] expose 优雅退出
- [ ] agent 优雅退出
