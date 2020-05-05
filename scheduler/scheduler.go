package scheduler

import (
    "time"
    "owlhmaster/utils"
    "owlhmaster/rulesetSource"
    "strconv"
    "owlhmaster/database"
    "owlhmaster/nodeclient"
    "owlhmaster/node"
    "github.com/astaxie/beego/logs"
)

func Init() {
    minutes, err := utils.GetKeyValueString("scheduler", "minutes")
    if err != nil {logs.Error("Scheduler Error getting data from main.conf: "+err.Error())}
    status, err := utils.GetKeyValueString("scheduler", "status")
    if err != nil {logs.Error("Scheduler Error getting data from main.conf: "+err.Error())}

    for status == "enabled"{
        RunScheduler()
        t,err := utils.GetKeyValueString("loop", "scheduler")
        if err != nil {logs.Error("Search Error: Cannot load scheduler information.")}
        tDuration, err := strconv.Atoi(t)
        for {            
            time.Sleep(time.Second * time.Duration(tDuration))
            _, currentMinutes, _ := time.Now().Clock()
            confMinutes,_ := strconv.Atoi(minutes)

            if currentMinutes % confMinutes == 0 {
                break
            }
        }
        logs.Info("Scheduler Running")
    }
}

//update task if their time is out
func RunScheduler() bool {    
    t := time.Now().Unix()
    currentTime := strconv.FormatInt(t, 10)
    tasks,err := CheckTasks()    
    if err != nil {logs.Error("Error RunScheduler checking tasks: %s", err.Error())}

    for j,k := range tasks {
        if k["nextEpoch"] <= currentTime{
            err = TaskUpdater(k)
            if err != nil {
                logs.Error("Error RunScheduler TaskUpdater: %s", err.Error())    
                continue
            }else{
            //calculate next epoch
            dbTime,_ := strconv.Atoi(k["period"])
            nextEpoch,_ := strconv.Atoi(k["nextEpoch"])
            s := strconv.Itoa(nextEpoch + dbTime)

            //update next epoch
            err = ndb.UpdateScheduler(j, "nextEpoch", s)
            if err != nil {
                logs.Error("Error RunScheduler UpdateScheduler updating next EPOCH time: %s", err.Error())    
                continue
            }
            logs.Notice("EPOCH updated")
            }
        }
    }
    return true
}

//return all the enabled tasks
func CheckTasks()(tasksEnabled map[string]map[string]string, err error){
    tasks,err := ndb.GetAllScheduler()
    if err != nil {logs.Error("Error CheckTasks GetAllScheduler: %s", err.Error());return nil,err}

    if len(tasks) == 0 { return nil,nil }
    enabledTasks := make(map[string]map[string]string)
    for x,y := range tasks{
        if tasks[x]["status"] == "enabled"{
            values := make(map[string]string)
            for y,_ := range y{
                    values[y] = tasks[x][y]
            }
            enabledTasks[x] = values
        }
    }
    return enabledTasks,nil
}

func SchedulerTask(content map[string]string)(err error){
    t := time.Now().Unix()
    currentTime := strconv.FormatInt(t, 10)
    taskUUID,err := ndb.GetSchedulerByValue(content["uuid"])
    if err!=nil { logs.Error("Error getting scheduler by value: "+err.Error()); return err}

    timeEpoch,err := utils.EpochTime(content["year"]+"-"+content["month"]+"-"+content["day"]+"T"+content["hour"]+":"+content["minute"]+":00")
    if err!=nil { logs.Error("Error getting rules from ruleset for update scheduler: "+err.Error()); return err}

    if taskUUID == "" {
        newUUID := utils.Generate()
        err = ndb.InsertScheduler(newUUID, "type", content["type"])
        err = ndb.InsertScheduler(newUUID, "update", content["update"])
        err = ndb.InsertScheduler(newUUID, "period", content["period"])
        err = ndb.InsertScheduler(newUUID, "uuid", content["uuid"])
        err = ndb.InsertScheduler(newUUID, "nextEpoch", strconv.FormatInt(timeEpoch, 10))
        err = ndb.InsertScheduler(newUUID, "status", content["status"])        
        if err != nil {logs.Error("Error SchedulerTask TaskUpdater after first update: %s", err.Error());return err}
        //INSERT LOG
        err = ndb.InsertSchedulerLog(content["uuid"], currentTime, "Task added. Next update = "+utils.HumanTime(timeEpoch)+". Update type = "+content["update"]+". Update period(in seconds) = "+content["period"]+". Status = "+content["status"])
        if err != nil {logs.Error("Error inserting Log: %s", err.Error());return err}

        logs.Notice("Task added")
    }else{
        err = ndb.UpdateScheduler(taskUUID, "status", "enabled")
        err = ndb.UpdateScheduler(taskUUID, "update", content["update"])
        err = ndb.UpdateScheduler(taskUUID, "period", content["period"])
        err = ndb.UpdateScheduler(taskUUID, "nextEpoch", strconv.FormatInt(timeEpoch, 10))
        if err != nil {logs.Error("Error UpdateScheduler task: %s", err.Error());return err}

        //INSERT LOG
        err = ndb.InsertSchedulerLog(content["uuid"], currentTime, "Task Updated. Next epoch = "+utils.HumanTime(timeEpoch)+". Update type = "+content["update"]+". Update period(in seconds) = "+content["period"]+". Status = "+content["status"])
        if err != nil {logs.Error("Error inserting Log: %s", err.Error());return err}
        logs.Notice("Task updated")
    }
    return nil
}

func StopTask(content map[string]string)(err error){

    t := time.Now().Unix()
    currentTime := strconv.FormatInt(t, 10)

    taskUUID,err := ndb.GetSchedulerByValue(content["uuid"])
    if err!=nil { logs.Error("Error stopping scheduler task: "+err.Error()); return err}

    err = ndb.UpdateScheduler(taskUUID, "status", "disabled")
    if err != nil {
        logs.Error("Error StopTask UpdateScheduler: %s", err.Error())
        //INSERT LOG
        err = ndb.InsertSchedulerLog(content["uuid"], currentTime, "Task ERROR: "+ err.Error())
        if err != nil {
            logs.Error("Error inserting Log: %s", err.Error())
            return err
        }
        return err
    }
    //INSERT LOG
    err = ndb.InsertSchedulerLog(content["uuid"], currentTime, "Task Updated: status == Disabled")
    if err != nil {logs.Error("Error inserting Log: %s", err.Error());return err}    
    return nil
}

func TaskUpdater(content map[string]string)(err error){
    t := time.Now().Unix()
    currentTime := strconv.FormatInt(t, 10)

    data,err := ndb.GetRulesFromRuleset(content["uuid"])
    if err!=nil { logs.Error("Error getting rules from ruleset for update scheduler: "+err.Error()); return err}
    for x := range data{
        values,err := ndb.GetRuleFilesByUniqueid(x)
        if err != nil {
            logs.Error("TimeSchedule Error GetRuleFilesByUniqueid values: %s", err)
            //INSERT LOG
            err = ndb.InsertSchedulerLog(content["uuid"], currentTime, "Task ERROR: "+ err.Error())
            if err != nil {
                logs.Error("Error inserting Log: %s", err.Error())
                return err
            }
            
            return err
        }
        for y := range values{
            sourceFile,err := ndb.GetRuleFilesByUniqueid(values[y]["sourceFileUUID"])
            if err != nil {
                logs.Error("TimeSchedule Error GetRuleFilesByUniqueid sourceFile: %s", err)
                //INSERT LOG
                err = ndb.InsertSchedulerLog(content["uuid"], currentTime, "Task ERROR: "+ err.Error())
                if err != nil {
                    logs.Error("Error inserting Log: %s", err.Error())
                    return err
                }
                return err
            }
            for z := range sourceFile{
                rulesetMap := make(map[string]string)
                sourceUUIDValue,err := ndb.GetRuleFilesValue(z,"sourceUUID")
                if err != nil {
                    logs.Error("TimeSchedule Error GetRuleFilesValue sourceUUIDValue: %s", err)
                    //INSERT LOG
                    err = ndb.InsertSchedulerLog(content["uuid"], currentTime, "Task ERROR: "+ err.Error())
                    if err != nil {
                        logs.Error("Error inserting Log: %s", err.Error())
                        return err
                    }
                    return err
                }
                finalData,err := ndb.GetAllDataRulesetDB(sourceUUIDValue)
                if err != nil {
                    logs.Error("TimeSchedule Error GetAllDataRulesetDB finalData: %s", err)
                    //INSERT LOG
                    err = ndb.InsertSchedulerLog(content["uuid"], currentTime, "Task ERROR: "+ err.Error())
                    if err != nil {
                        logs.Error("Error inserting Log: %s", err.Error())
                        return err
                    }
                    return err
                }
                for a,b := range finalData{
                    for b,_ := range b {
                        rulesetMap[b] = finalData[a][b]
                    }
                }
                if rulesetMap["sourceType"] == "custom"{continue}

                if rulesetMap["isDownloaded"] == "false"{
                    err = rulesetSource.DownloadFile(rulesetMap)
                    if err != nil {
                        logs.Error("TimeSchedule Error Downloading: %s", err)
                        //INSERT LOG
                        err = ndb.InsertSchedulerLog(content["uuid"], currentTime, "Task ERROR Downloading: "+ err.Error())
                        if err != nil {
                            logs.Error("Error inserting Log: %s", err.Error())
                            return err
                        }
                        return err
                    }
                }else if rulesetMap["isDownloaded"] == "true"{
                    // rulesetMap["uuid"] = content["uuid"]
                    err = rulesetSource.OverwriteDownload(rulesetMap)
                    if err != nil {
                        logs.Error("TimeSchedule Error Overwriting: %s", err)
                        //INSERT LOG
                        err = ndb.InsertSchedulerLog(content["uuid"], currentTime, "Task ERROR Overwriting: "+ err.Error())
                        if err != nil {
                            logs.Error("Error inserting Log: %s", err.Error())
                            return err
                        }
                        return err
                    }                            
                }    
            }
        }                
    }
    //overwrite files for this ruleset
    for d := range data{
        if content["update"] == "overwrite" {
            err = rulesetSource.OverwriteRuleFile(d)
            if err != nil {
                logs.Error("TimeSchedule Error OverwriteRuleFile ruleset: %s", err)
                //INSERT LOG
                err = ndb.InsertSchedulerLog(content["uuid"], currentTime, "Task ERROR Overwriting file content: "+ err.Error())
                if err != nil {
                    logs.Error("Error inserting Log: %s", err.Error())
                    return err
                }
                return err
            }
        }else if content["update"] == "add-lines" {
            err = rulesetSource.AddNewLinesToRuleset(d)
            if err != nil {
                logs.Error("TimeSchedule Error AddNewLinesToRuleset ruleset: %s", err)
                //INSERT LOG
                err = ndb.InsertSchedulerLog(content["uuid"], currentTime, "Task ERROR adding new lines to file content: "+ err.Error())
                if err != nil {
                    logs.Error("Error inserting Log: %s", err.Error())
                    return err
                }
                return err
            }
        }
    }

    //synchronize to all nodes
        //get all nodes with this ruleset
        nodeList,err := ndb.GetNodeWithRulesetUUID(content["uuid"])
        if err != nil {logs.Error("SCHEDULER Error getting nodes by ruleset: %s", err.Error()); return err}
        //get all nodes into a group with this ruleset
        //get all groups
        allGroups,err := ndb.GetAllGroups()
        if err != nil {logs.Error("SCHEDULER Error getting all groups: %s", err.Error()); return err}
        allGroupNodes,err := ndb.GetAllGroupNodes()
        if err != nil {logs.Error("SCHEDULER Error getting all groupNodes: %s", err.Error()); return err}
        rulesetName,err := ndb.GetRulesetSourceValue(content["uuid"], "name")

        for x := range allGroups{
            //get group id if their ruleset is our ruleset
            if allGroups[x]["rulesetID"] == content["uuid"] {
                // get all nodes into this group
                for y := range allGroupNodes{
                    if allGroupNodes[y]["groupid"] == x {
                        //check if node is into array and push
                        exists := false
                        for node := range nodeList {
                            if nodeList[node] == allGroupNodes[y]["nodesid"] {
                                exists = true
                            }
                        }
                        if !exists {
                            nodeList = append(nodeList, allGroupNodes[y]["nodesid"])
                        }
                    }
                }
            }
        }

        for nodeID := range nodeList {
            values := make(map[string][]byte)
            //ruleset id --> content["uuid"]

            //get node token
            err = ndb.GetTokenByUuid(nodeList[nodeID]); if err!=nil{logs.Error("scheduler/TaskUpdater Error loading node token: %s",err); return err}
            //get node ip and port
            ipnid,portnid,err := ndb.ObtainPortIp(nodeList[nodeID])
            if err != nil { logs.Error("scheduler/TaskUpdater ERROR Obtaining Port and Ip: "+err.Error()); return err}

            //get ruleset content
            rulesetData,err := node.CreateNewRuleFile(content["uuid"])
            if err != nil {logs.Error("scheduler/TaskUpdater error creating ruleset file: "+err.Error()); return err}
            
            values["data"] = rulesetData
            values["name"] = []byte(rulesetName)

            //send to 
            err = nodeclient.SyncGroupRulesetToNode(ipnid, portnid, values)
            if err != nil {logs.Error("scheduler/TaskUpdater error SyncGroupRulesetToNode: "+err.Error()); return err}

        }
    

    // err = node.SyncRulesetToAllNodes(content)
    // if err != nil {
    //     logs.Error("TimeSchedule Error synchronizing ruleset: %s", err)
    //     err = ndb.InsertSchedulerLog(content["uuid"], currentTime, "Task ERROR synchronizing: "+ err.Error())
    //     if err != nil {
    //         logs.Error("Error inserting Log: %s", err.Error())
    //         return err
    //     }
    //     return err
    // }

    err = ndb.InsertSchedulerLog(content["uuid"], currentTime, "Task synchronized for ruleset "+content["uuid"])
    if err != nil {
        logs.Error("Error inserting Log: %s", err.Error())
        return err
    }
    logs.Notice("Ruleset synchronized "+content["uuid"])    
    return nil
}

func GetLog(uuid string) (logReg map[string]map[string]string ,err error) {
    logValue,err := ndb.GetSchedulerLogByUniqueid(uuid)
    if err != nil {
        logs.Error("Error getting LOG: %s", err.Error())
        return nil,err
    }
    return logValue,nil
}