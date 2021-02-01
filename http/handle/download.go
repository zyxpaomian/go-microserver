package handle

import (
	"microserver/common"
	log "microserver/common/formatlog"
    cfg "microserver/common/configparse"
	"net/http"
    "os"
    "strconv"
    "io"
    "fmt"
)

// 下载新版本的agent
func packageNewAgent(res http.ResponseWriter, req *http.Request) {
    newAgent, err := os.Open(cfg.GlobalConf.GetStr("package", "newagent"))
    if err != nil {
        log.Errorf("[http] packageNewAgent 下载文件打开失败, %v", err.Error())
		common.ResMsg(res, 500, err.Error())
		return        
    }

    fileHeader := make([]byte, 512)
    newAgent.Read(fileHeader)

    fileStat, _ := newAgent.Stat()
    fileDstName := fmt.Sprintf("attachment; filename=%s", cfg.GlobalConf.GetStr("package", "newagent"))
    res.Header().Set("Content-Disposition", fileDstName)
    res.Header().Set("Content-Type", http.DetectContentType(fileHeader))
    res.Header().Set("Content-Length", strconv.FormatInt(fileStat.Size(), 10))
    newAgent.Seek(0, 0)
    io.Copy(res, newAgent)
    return
}

