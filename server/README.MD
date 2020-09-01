Build USDZ Server
```
docker build -t usdzserver .
```

Run USDZ Server
```
docker run --env PORT=8080 --env APP_STATIC_PATH=./static --env MODELS_PATH=./models -p 8080:8080 --name usd1 usdzserver
```

Test the endpoint
```
# convert obj
curl -X POST 'https://ar-kitect.herokuapp.com?mode=obj' -F f.obj=@'somefilepath/file.obj'
# convert fbx
curl -X POST 'https://ar-kitect.herokuapp.com?mode=fbx' -F f.obj=@'somefilepath/file.fbx'
```
