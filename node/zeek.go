package node

import (
    "github.com/astaxie/beego/logs"
// 	  "strings"
//    "database/sql"
//    "fmt"
//    "time"
//    _ "github.com/mattn/go-sqlite3"
    "crypto/tls"
    "owlhmaster/database"
    "errors"
    "owlhmaster/nodeclient"
    "owlhmaster/utils"
    // "encoding/json"
//    "regexp"
    "io/ioutil"
    "net/http"
    // "bytes"
)

func Zeek(n string) (data map[string]bool, err error) {
    logs.Info("Node Zeek -> IN")

    ip,port,err := utils.ObtainPortIp(n)
    if err != nil {
        logs.Info("Zeek - get IP and PORT Error -> %s", err.Error())
        return nil,err
    }    
    logs.Info("Zeek IP and PORT -> %s, %s", ip, port)
    data, err = nodeclient.Zeek(ip,port)
    if err != nil {
        return nil,err
    }
    return data,nil
}

func RunZeek(uuid string)(data string, err error){
    if ndb.Db == nil {
        logs.Error("RunZeek -- Can't acces to database")
        return "", errors.New("RunZeek -- Can't acces to database")
    }
    
    // ipnid,portnid,err := GetSuricataIpPort(uuid)
    ipnid,portnid,err := utils.ObtainPortIp(uuid)
    
    url := "https://"+ipnid+":"+portnid+"/node/zeek/RunZeek"
    req, err := http.NewRequest("PUT", url, nil)
    tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true},}
    client := &http.Client{Transport: tr}
    resp, err := client.Do(req)

    if err != nil {
        return "",err
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
    logs.Info("RunZeek function "+string(body))
    return string(body),nil
}

func StopZeek(uuid string)(data string, err error){
    if ndb.Db == nil {
        logs.Error("StopZeek -- Can't acces to database")
        return "", errors.New("StopZeek -- Can't acces to database")
    }

    // ipnid,portnid,err := GetSuricataIpPort(uuid)
    ipnid,portnid,err := utils.ObtainPortIp(uuid)

    url := "https://"+ipnid+":"+portnid+"/node/zeek/StopZeek"
    req, err := http.NewRequest("PUT", url, nil)
    tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true},}
    client := &http.Client{Transport: tr}
    resp, err := client.Do(req)

    if err != nil {
        return "",err
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
    return string(body),nil
}