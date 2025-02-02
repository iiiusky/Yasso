package cmd

import (
	"Yasso/config"
	"fmt"
	"github.com/spf13/cobra"
	"sync"
	"time"
)

/*
	全体扫描模块
	将逐步执行每个任务
*/

func printHeader(){

}



var allCmd = &cobra.Command{
	Use: "all",
	Short: "Use all scanner module (.attention)\nSome service not support proxy,You might lose it [*]",
	Run: func(cmd *cobra.Command, args []string) {
		if Hosts == "" {
			_ = cmd.Help()
			return
		}

		allRun(Hosts,Ports,LogBool,Runtime,PingBool)
		return
	},
}

func init(){
	allCmd.Flags().StringVarP(&Hosts,"host","H","","Set `hosts`(The format is similar to Nmap)")
	allCmd.Flags().StringVarP(&Ports,"ports","P","","Set `ports`(The format is similar to Nmap)")
	allCmd.Flags().BoolVar(&PingBool,"noping",true,"No use ping to scanner alive host")
	allCmd.Flags().IntVar(&Runtime,"runtime",100,"Set scanner ants pool thread")
	allCmd.Flags().StringVar(&ProxyHost,"proxy","","Set socks5 proxy")
	allCmd.Flags().DurationVar(&TimeDuration,"time",1 * time.Second,"Set timeout ")
	rootCmd.AddCommand(allCmd)
}

func allRun(hostString string,portString string,log bool,runtime int,noping bool){
	// 先对程序进行ping扫描
	// 解析hosts
	var (
		ips []string
		ports []int
		webports []int
		alive []string
		wg sync.WaitGroup
	)
	if hostString != "" {
		ips,_ = ResolveIPS(hostString) // 解析ip并获取ip列表
	}
	if Ports != "" {
		ports ,_ = ResolvePORTS(portString)
		webports,_ = ResolvePORTS(config.DisMapPorts)
	}else{
		ports ,_ = ResolvePORTS(DefaultPorts)
		webports,_ = ResolvePORTS(config.DisMapPorts)
	}

	if noping == true {
		// 不执行ping操作
		alive = ips
	}else{
		// 执行 ping 操作
		fmt.Println("----- [Yasso] Start do ping scan -----")
		alive = execute(ips)
	}
	// 做漏洞扫描
	if len(alive) > 0 {
		fmt.Println("----- [Yasso] Start do vuln scan -----")
		VulScan(alive,false,true,false) // 做全扫描
		fmt.Println("----- [Yasso] Start do port scan -----")
		PortResults := PortScan(alive,ports)
		// 获取我们的端口扫描结果，去遍历
		if len(PortResults) != 0 {
			fmt.Println("----- [Yasso] Start do crack service -----")
			for _,v := range PortResults {
				// 这里做漏洞扫描
				wg.Add(1)
				go func(v PortResult) {
					defer wg.Done()
					for _,p := range v.Port {
						switch p {
						case 21:
							users,pass := ReadTextToDic("ftp",UserDic,PassDic)
							burpTask(v.IP,"ftp",users,pass,p,100,1*time.Second,"",false)
						case 22:
							users,pass := ReadTextToDic("ssh",UserDic,PassDic)
							burpTask(v.IP,"ssh",users,pass,p,100,1*time.Second,"",false)
						case 3306:
							users,pass := ReadTextToDic("mysql",UserDic,PassDic)
							burpTask(v.IP,"mysql",users,pass,p,100,1*time.Second,"",false)
						case 6379:
							_,_,_ = RedisUnAuthConn(config.HostIn{Host: v.IP,Port: p,TimeOut:1 * time.Second},"test","test")
							users,pass := ReadTextToDic("redis",UserDic,PassDic)
							burpTask(v.IP,"redis",users,pass,p,100,5*time.Second,"",false)
						case 1433:
							users,pass := ReadTextToDic("mssql",UserDic,PassDic)
							burpTask(v.IP,"mssql",users,pass,p,100,1*time.Second,"",false)
						case 5432:
							users,pass := ReadTextToDic("postgres",UserDic,PassDic)
							burpTask(v.IP,"postgres",users,pass,p,100,1*time.Second,"",false)
						case 27017:
							_,_ = MongoUnAuth(config.HostIn{Host: v.IP,Port: p,TimeOut: 1*time.Second},"test","test")
							users,pass := ReadTextToDic("mongodb",UserDic,PassDic)
							burpTask(v.IP,"mongodb",users,pass,p,100,1*time.Second,"",false)
						case 445:
							users,pass := ReadTextToDic("smb",UserDic,PassDic)
							burpTask(v.IP,"smb",users,pass,p,100,1*time.Second,"",false)
						case 5985:
							users,pass := ReadTextToDic("rdp",UserDic,PassDic) // winrm与本地rdp认证相同
							burpTask(v.IP,"winrm",users,pass,p,100,1*time.Second,"",false)
						}
					}
				}(v)
			}
			wg.Wait()
		}
		// 做网卡扫描
		fmt.Println("----- [Yasso] Start do Windows service scan -----")
		winscan(alive,true)
		fmt.Println("----- [Yasso] Start do web service scan -----")
		DisMapScan(alive,webports)
	}
}