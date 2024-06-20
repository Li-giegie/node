$filteredArray = @("example*","register_service*")
# 获取当前目录的路径
$currentDir = Get-Item -Path ".\"
$currentDirPath = $currentDir.FullName

# 递归获取当前目录及其所有子目录中的目录对象
$directories = Get-ChildItem -Path $currentDirPath -Recurse -Directory | Where-Object { $_.Name -notlike '.*' }

# 循环遍历目录对象并输出基于当前目录为前缀的目录名
foreach ($directory in $directories) {
   # 使用Replace方法替换完整路径中的当前目录路径，得到基于当前目录为前缀的相对路径
   $relativePath = $directory.FullName.Replace($currentDirPath, "").TrimStart("\")
   $relativePath = $relativePath.Replace("\", "/")
#    echo $relativePath
    # 定义一个标志变量，用来记录是否找到匹配项
    $foundMatch = $false
    # 遍历数组b中的每个元素
    foreach ($pattern in $filteredArray) {
        # 检查变量a是否匹配当前的模式
        if ($relativePath -like $pattern) {
            $foundMatch = $true
            break # 找到匹配项后退出循环
        }
    }
    # 根据标志变量输出结果
    if (-not $foundMatch) {
         $packageName = "github.com/Li-giegie/node/"+$relativePath
         go vet $packageName ;echo $packageName
    }
}
