# [ar-kitect](https://ar-kitect.herokuapp.com/)
3D AR Model Viewer &amp; Exporter

**Usage**
```
# convert obj
curl -X POST 'https://ar-kitect.herokuapp.com?mode=obj' -F f.obj=@'somefilepath/file.obj'
# convert fbx
curl -X POST 'https://ar-kitect.herokuapp.com?mode=fbx' -F f.obj=@'somefilepath/file.fbx'
```
