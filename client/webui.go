package main

import (
	"fmt"
	"time"
	"sort"
	"strings"
	"net/http"
	"io/ioutil"
	"github.com/piotrnar/gocoin/btc"
)

var webuimenu = [][2]string {
	{"/", "Home"},
	{"/net", "Network"},
	{"/txs", "Transactions"},
	{"/blocks", "Blocks"},
	{"/miners", "Miners"},
	{"/counts", "Counters"},
}

const htmlhead = `<script type="text/javascript" src="webui/gocoin.js"></script>
<link rel="stylesheet" href="webui/gocoin.css" type="text/css"></head><body>
<table align="center" width="990" cellpadding="0" cellspacing="0"><tr><td>
`


func p_webui(w http.ResponseWriter, r *http.Request) {
	if len(strings.SplitN(r.URL.Path[1:], "/", 3))==2 {
		dat, _ := ioutil.ReadFile(r.URL.Path[1:])
		w.Write(dat)
	}
}

func write_html_head(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("<html><head><title>Gocoin "+btc.SourcesTag+"</title>\n"))
	w.Write([]byte(htmlhead))
	for i := range webuimenu {
		if i==len(webuimenu)-1 {
			w.Write([]byte("<td align=\"right\">")) // Last menu item on the right
		} else if i==0 {
			w.Write([]byte("<table width=\"100%\"><tr><td>"))
		} else {
			w.Write([]byte(" | "))
		}
		w.Write([]byte("<a "))
		if r.URL.Path==webuimenu[i][0] {
			w.Write([]byte("class=\"menuat\" "))
		}
		w.Write([]byte("href=\""+webuimenu[i][0]+"\">"+webuimenu[i][1]+"</a>"))
	}
	w.Write([]byte("</table><hr>\n"))
}

func write_html_tail(w http.ResponseWriter) {
	w.Write([]byte("</body></html>"))
}

func p_home(w http.ResponseWriter, r *http.Request) {
	write_html_head(w, r)

	fmt.Fprint(w, "<h2>Wallet</h2>")
	fmt.Fprintf(w, "Last known balance: <b>%.8f</b> BTC in <b>%d</b> outputs\n",
		float64(LastBalance)/1e8, len(MyBalance))
	fmt.Fprintln(w, " - <input type=\"button\" value=\"Show\" onclick=\"raw_load('balance', 'Unspent outputs')\">")

	fmt.Fprint(w, "<h2>Last Block</h2>")
	mutex.Lock()
	fmt.Fprintln(w, "<table>")
	fmt.Fprintf(w, "<tr><td>Hash:<td><b>%s</b>\n", LastBlock.BlockHash.String())
	fmt.Fprintf(w, "<tr><td>Height:<td><b>%d</b>\n", LastBlock.Height)
	fmt.Fprintf(w, "<tr><td>Timestamp:<td><b>%s</b>\n",
		time.Unix(int64(LastBlock.Timestamp), 0).Format("2006/01/02 15:04:05"))
	fmt.Fprintf(w, "<tr><td>Difficulty:<td><b>%.3f</b>\n", btc.GetDifficulty(LastBlock.Bits))
	fmt.Fprintf(w, "<tr><td>Received:<td><b>%s</b> ago\n", time.Now().Sub(LastBlockReceived).String())
	fmt.Fprintln(w, "</table>")
	mutex.Unlock()

	fmt.Fprintln(w, "<br><table><tr><td valign=\"top\">")

	fmt.Fprint(w, "<h2>Network</h2>")
	fmt.Fprintln(w, "<table>")
	bw_mutex.Lock()
	tick_recv()
	tick_sent()
	fmt.Fprintf(w, "<tr><td>Downloading at:<td><b>%d/%d</b> KB/s, <b>%s</b> total\n",
		dl_bytes_prv_sec>>10, DownloadLimit>>10, bts(dl_bytes_total))
	fmt.Fprintf(w, "<tr><td>Uploading at:<td><b>%d/%d</b> KB/s, <b>%s</b> total\n",
		ul_bytes_prv_sec>>10, UploadLimit>>10, bts(ul_bytes_total))
	bw_mutex.Unlock()
	fmt.Fprintf(w, "<tr><td>Net Block Queue Size:<td><b>%d</b>\n", len(netBlocks))
	fmt.Fprintf(w, "<tr><td>Net Tx Queue Size:<td><b>%d</b>\n", len(netTxs))
	fmt.Fprintf(w, "<tr><td>Open Connections:<td><b>%d</b> (<b>%d</b> outgoing + <b>%d</b> incomming)\n",
		len(openCons), OutConsActive, InConsActive)
	fmt.Fprint(w, "<tr><td>Extrenal IPs:<td>")
	for ip, cnt := range ExternalIp4 {
		fmt.Fprintf(w, "%d.%d.%d.%d (%d)&nbsp;&nbsp;", byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip), cnt)
	}
	fmt.Fprintln(w, "</table>")

	fmt.Fprintln(w, "<td wifth=\"100\"><td valign=\"top\">")

	fmt.Fprint(w, "<h2>Others</h2>")
	fmt.Fprintln(w, "<table>")
	fmt.Fprintf(w, "<tr><td>Blocks Cached:<td><b>%d</b>\n", len(cachedBlocks))
	fmt.Fprintf(w, "<tr><td>Blocks Pending:<td><b>%d/%d</b>\n", len(pendingBlocks), len(pendingFifo))
	fmt.Fprintf(w, "<tr><td>Known Peers:<td><b>%d</b>\n", peerDB.Count())
	fmt.Fprintf(w, "<tr><td>Node's uptime:<td><b>%s</b>\n", time.Now().Sub(StartTime).String())

	fmt.Fprintln(w, "</table>")

	w.Write([]byte("</table><br><h2 id=\"rawtit\"></h2><pre id=\"rawdiv\" class=\"mono\"></pre>"))
	write_html_tail(w)
}

func p_net(w http.ResponseWriter, r *http.Request) {
	write_html_head(w, r)
	mutex.Lock()
	srt := make(sortedkeys, len(openCons))
	cnt := 0
	for k, v := range openCons {
		srt[cnt].key = k
		srt[cnt].ConnID = v.ConnID
		cnt++
	}
	sort.Sort(srt)
	fmt.Fprintf(w, "<b>%d</b> outgoing and <b>%d</b> incomming connections<br><br>\n", OutConsActive, InConsActive)
	fmt.Fprintln(w, "<table class=\"netcons\" border=\"1\" cellspacing=\"0\" cellpadding=\"0\">")
	fmt.Fprint(w, "<tr><th>ID<th colspan=\"2\">IP<th>Ping<th colspan=\"2\">Last Rcvd<th colspan=\"2\">Last Sent")
	fmt.Fprintln(w, "<th>Total Rcvd<th>Total Sent<th colspan=\"2\">Version<th>Sending")
	for idx := range srt {
		v := openCons[srt[idx].key]
		fmt.Fprintf(w, "<tr class=\"hov\" style=\"cursor:pointer;\" onclick=\"raw_load('net?id=%d', 'Connection')\"><td align=\"right\">%d",
			v.ConnID, v.ConnID)
		if v.Incomming {
			fmt.Fprint(w, "<td aling=\"center\">From")
		} else {
			fmt.Fprint(w, "<td aling=\"center\">To")
		}
		fmt.Fprint(w, "<td align=\"right\">", v.PeerAddr.Ip())
		fmt.Fprint(w, "<td align=\"right\">", v.GetAveragePing(), "ms")
		fmt.Fprint(w, "<td align=\"right\">", v.LastBtsRcvd)
		fmt.Fprint(w, "<td class=\"mono\">", v.LastCmdRcvd)
		fmt.Fprint(w, "<td align=\"right\">", v.LastBtsSent)
		fmt.Fprint(w, "<td class=\"mono\">", v.LastCmdSent)
		fmt.Fprint(w, "<td align=\"right\">", bts(v.BytesReceived))
		fmt.Fprint(w, "<td align=\"right\">", bts(v.BytesSent))
		fmt.Fprint(w, "<td align=\"right\">", v.node.version)
		fmt.Fprint(w, "<td>", v.node.agent)
		fmt.Fprintf(w, "<td align=\"right\">%d/%d", v.send.sofar, len(v.send.buf))
	}
	w.Write([]byte("</table><br><h2 id=\"rawtit\"></h2><pre id=\"rawdiv\" class=\"mono\"></pre>"))
	mutex.Unlock()
	write_html_tail(w)
}

func p_txs(w http.ResponseWriter, r *http.Request) {
	write_html_head(w, r)
	tx_mutex.Lock()
	fmt.Fprintln(w, "<table>")

	fmt.Fprint(w, "<tr><td>Transactions To Send:<td>")
	fmt.Fprintln(w, "<input type=\"button\" value=\"", len(TransactionsToSend),
		"\" onclick=\"raw_load('txs2s', 'Transactions To Send')\">")

	fmt.Fprint(w, "<tr><td>Rejected Transactions:<td>")
	fmt.Fprintln(w, "<input type=\"button\" value=\"", len(TransactionsRejected),
		"\" onclick=\"raw_load('txsre', 'Rejected Transactions')\">")

	fmt.Fprintf(w, "<tr><td>Pending Transactions:<td><b>%d</b> / <b>%d</b>\n", len(TransactionsPending), len(netTxs))
	tx_mutex.Unlock()

	w.Write([]byte("</table><br><h2 id=\"rawtit\"></h2><pre id=\"rawdiv\" class=\"mono\"></pre>"))
	write_html_tail(w)
}

func p_blocks(w http.ResponseWriter, r *http.Request) {
	write_html_head(w, r)
	end := BlockChain.BlockTreeEnd
	fmt.Fprint(w, "<table border=\"1\" cellspacing=\"0\" cellpadding=\"0\">\n")
	fmt.Fprintf(w, "<tr><th>Height<th>Timestamp<th>Hash<th>Txs<th>Size<th>Mined by</tr>\n")
	for cnt:=0; end!=nil && cnt<100; cnt++ {
		bl, _, e := BlockChain.Blocks.BlockGet(end.BlockHash)
		if e != nil {
			return
		}
		block, e := btc.NewBlock(bl)
		if e!=nil {
			return
		}
		block.BuildTxList()
		miner := blocks_miner(bl)
		fmt.Fprintf(w, "<tr class=\"hov\"><td>%d<td>%s", end.Height,
			time.Unix(int64(block.BlockTime), 0).Format("2006-01-02 15:04:05"))
		fmt.Fprintf(w, "<td><a class=\"mono\" href=\"http://blockchain.info/block/%s\">%s",
			end.BlockHash.String(), end.BlockHash.String())
		fmt.Fprintf(w, "<td align=\"right\">%d<td align=\"right\">%d<td align=\"center\">%s</tr>\n",
			len(block.Txs), len(bl), miner)
		end = end.Parent
	}
	fmt.Fprint(w, "</table>")
	write_html_tail(w)
}

type onemiernstat []struct {
	name string
	cnt int
}

func (x onemiernstat) Len() int {
	return len(x)
}

func (x onemiernstat) Less(i, j int) bool {
	return x[i].cnt > x[j].cnt
}

func (x onemiernstat) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}

func p_miners(w http.ResponseWriter, r *http.Request) {
	write_html_head(w, r)
	m := make(map[string]int, 20)
	cnt, unkn := 0, 0
	end := BlockChain.BlockTreeEnd
	var lastts uint32
	now := uint32(time.Now().Unix())
	for ; end!=nil; cnt++ {
		if now-end.Timestamp > 24*3600 {
			break
		}
		lastts = end.Timestamp
		bl, _, e := BlockChain.Blocks.BlockGet(end.BlockHash)
		if e != nil {
			return
		}
		miner := blocks_miner(bl)
		if miner!="" {
			m[miner]++
		} else {
			unkn++
		}
		end = end.Parent
	}
	srt := make(onemiernstat, len(m))
	i := 0
	for k, v := range m {
		srt[i].name = k
		srt[i].cnt = v
		i++
	}
	sort.Sort(srt)
	fmt.Fprintf(w, "Data from last <b>%d</b> blocks, starting at <b>%s</b><br><br>\n",
		cnt, time.Unix(int64(lastts), 0).Format("2006-01-02 15:04:05"))
	fmt.Fprint(w, "<table border=\"1\" cellspacing=\"0\" cellpadding=\"0\">\n")
	fmt.Fprint(w, "<tr><th>Miner<th>Blocks<th>Share</tr>\n")
	for i := range srt {
		fmt.Fprintf(w, "<tr class=\"hov\"><td>%s<td align=\"right\">%d<td align=\"right\">%.0f%%</tr>\n",
			srt[i].name, srt[i].cnt, 100*float64(srt[i].cnt)/float64(cnt))
	}
	fmt.Fprintf(w, "<tr class=\"hov\"><td><i>Unknown</i><td align=\"right\">%d<td align=\"right\">%.0f%%</tr>\n",
		unkn, 100*float64(unkn)/float64(cnt))
	fmt.Fprint(w, "</table><br>")
	fmt.Fprintf(w, "Average blocks per hour: <b>%.2f</b>", float64(cnt)/(float64(now-lastts)/3600))
	write_html_tail(w)
}

func p_counts(w http.ResponseWriter, r *http.Request) {
	write_html_head(w, r)
	fmt.Fprint(w, "<h1>Counters</h1>")
	counter_mutex.Lock()
	ck := make([]string, 0)
	for k, _ := range Counter {
		ck = append(ck, k)
	}
	sort.Strings(ck)
	fmt.Fprint(w, "<table class=\"mono\"><tr>")
	fmt.Fprint(w, "<td valign=\"top\"><table border=\"1\"><tr><th colspan=\"2\">Generic Counters")
	prv_ := ""
	for i := range ck {
		if ck[i][4]=='_' {
			if ck[i][:4]!=prv_ {
				prv_ = ck[i][:4]
				fmt.Fprint(w, "</table><td valign=\"top\"><table border=\"1\"><tr><th colspan=\"2\">")
				switch prv_ {
					case "rbts": fmt.Fprintln(w, "Received bytes")
					case "rcvd": fmt.Fprintln(w, "Received messages")
					case "sbts": fmt.Fprintln(w, "Sent bytes")
					case "sent": fmt.Fprintln(w, "Sent messages")
					default: fmt.Fprintln(w, prv_)
				}
			}
			fmt.Fprintf(w, "<tr><td>%s</td><td>%d</td></tr>\n", ck[i][5:], Counter[ck[i]])
		} else {
			fmt.Fprintf(w, "<tr><td>%s</td><td>%d</td></tr>\n", ck[i], Counter[ck[i]])
		}
	}
	fmt.Fprint(w, "</table></table>")
	counter_mutex.Unlock()
	write_html_tail(w)
}

func raw_balance(w http.ResponseWriter, r *http.Request) {
	for i := range MyBalance {
		fmt.Fprintf(w, "%7d %s\n", 1+BlockChain.BlockTreeEnd.Height-MyBalance[i].MinedAt,
			MyBalance[i].String())
	}
}

func raw_net(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(w, "Error")
		}
	}()

	ididx := strings.Index(r.RequestURI, "?id=")
	if ididx==-1 {
		fmt.Fprintln(w, "Missing id")
		return
	}
	v := look2conn(r.RequestURI[ididx+4:])
	if v == nil {
		fmt.Fprintln(w, "There is no such an active connection")
	}

	fmt.Fprintf(w, "Connection ID %d:\n", v.ConnID)
	if v.Incomming {
		fmt.Fprintln(w, "Comming from", v.PeerAddr.Ip())
	} else {
		fmt.Fprintln(w, "Going to", v.PeerAddr.Ip())
	}
	if !v.ConnectedAt.IsZero() {
		fmt.Fprintln(w, "Connected at", v.ConnectedAt.Format("2006-01-02 15:04:05"))
		if v.node.version!=0 {
			fmt.Fprintln(w, "Node Version:", v.node.version)
			fmt.Fprintln(w, "User Agent:", v.node.agent)
			fmt.Fprintln(w, "Chain Height:", v.node.height)
		}
		fmt.Fprintln(w, "Last data got/sent:", time.Now().Sub(v.LastDataGot).String())
		fmt.Fprintln(w, "Last command received:", v.LastCmdRcvd, " ", v.LastBtsRcvd, "bytes")
		fmt.Fprintln(w, "Last command sent:", v.LastCmdSent, " ", v.LastBtsSent, "bytes")
		fmt.Fprintln(w, "Bytes received:", v.BytesReceived)
		fmt.Fprintln(w, "Bytes sent:", v.BytesSent)
		fmt.Fprintln(w, "Next getbocks sending in", v.NextBlocksAsk.Sub(time.Now()).String())
		if v.LastBlocksFrom != nil {
			fmt.Fprintln(w, "Last block asked:", v.LastBlocksFrom.Height, v.LastBlocksFrom.BlockHash.String())
		}
		fmt.Fprintln(w, "Ticks:", v.TicksCnt, " Loops:", v.LoopCnt)
		if v.send.buf != nil {
			fmt.Fprintln(w, "Bytes to send:", len(v.send.buf), "-", v.send.sofar)
		}
		if len(v.PendingInvs)>0 {
			fmt.Fprintln(w, "Invs to send:", len(v.PendingInvs))
		}

		if v.GetBlockInProgress != nil {
			fmt.Fprintln(w, "GetBlock In Progress:", v.GetBlockInProgress.String())
		}

		// Display ping stats
		w.Write([]byte("Ping history:"))
		idx := v.PingHistoryIdx
		for _ = range(v.PingHistory) {
			fmt.Fprint(w, " ", v.PingHistory[idx])
			idx = (idx+1)%PingHistoryLength
		}
		fmt.Fprintln(w, " ->", v.GetAveragePing(), "ms")
	} else {
		fmt.Fprintln(w, "Not yet connected")
	}
}


func raw_txs2s(w http.ResponseWriter, r *http.Request) {
	cnt := 0
	tx_mutex.Lock()
	for k, v := range TransactionsToSend {
		cnt++
		var oe, snt string
		if v.own {
			oe = "OWN"
		} else {
			oe = "ext"
		}

		if v.sentCount==0 {
			snt = "never sent"
		} else {
			snt = fmt.Sprintf("sent %d times, last %s ago", v.sentCount,
				time.Now().Sub(v.lastTime).String())
		}
		fmt.Fprintf(w, "%5d) %s: %s - %d bytes - %s\n", cnt, oe,
			btc.NewUint256(k[:]).String(), len(v.data), snt)
	}
	tx_mutex.Unlock()
}


func raw_txsre(w http.ResponseWriter, r *http.Request) {
	cnt := 0
	tx_mutex.Lock()
	for k, v := range TransactionsRejected {
		cnt++
		fmt.Fprintf(w, "%5d) %s - %s ago\n", cnt, btc.NewUint256(k[:]).String(),
			time.Now().Sub(v).String())
	}
	tx_mutex.Unlock()
}


func webserver() {
	http.HandleFunc("/webui/", p_webui)
	http.HandleFunc("/", p_home)
	http.HandleFunc("/net", p_net)
	http.HandleFunc("/txs", p_txs)
	http.HandleFunc("/blocks", p_blocks)
	http.HandleFunc("/miners", p_miners)
	http.HandleFunc("/counts", p_counts)

	http.HandleFunc("/raw_txs2s", raw_txs2s)
	http.HandleFunc("/raw_txsre", raw_txsre)
	http.HandleFunc("/raw_balance", raw_balance)
	http.HandleFunc("/raw_net", raw_net)

	http.ListenAndServe(*webui, nil)
}
