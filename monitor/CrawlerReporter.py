#!/usr/bin/python
# coding:utf-8

import time
import traceback
import httplib
import sys
import os

#maillist = "jingleizhang@creditease.cn,liangxiang@creditease.cn,weiguozhu@creditease.cn,chunjialiu@creditease.cn"
maillist = "jingleizhang@creditease.cn"

graphite_url = "dmz.monitor"

crawlerUsage = {}

#办公网
#host = 'dmz.monitor.com'

#线上环境
host = 'dmz.monitor'

port = '80'

def getGraphiteRaw(url):
    conn = httplib.HTTPConnection(host + ":" + port)
    headers = {"accept":"text/plain", "ua":"python_httplib"}
    conn.request("GET", url, "", headers)
    
    resp = conn.getresponse()
    data = resp.read()
    print "response: ", resp.status, resp.reason, data
    
    sdata = data.split(",")
    conn.close()
    if sdata[len(sdata)-1] == "None":
        return sdata[len(sdata)-2]
    else:
        sdata[len(sdata)-1]


def getUsage():
    try:
        global crawlerUsage
        crawlerUsage.clear()
        
        url = '/render?from=-10minutes&until=now&rawData=True&target=sumSeries(stats.gauges.crawler.downloader.s2g*.8100.writePageCount)'
        crawlerUsage['totalwpcount'] = getGraphiteRaw(url)
        
        url = '/render?from=-10minutes&until=now&rawData=True&target=sumSeries(stats.gauges.crawler.downloader.s2g*.8100.totalDownloadedPageCount)'
        crawlerUsage['totaldpcount'] = getGraphiteRaw(url) 
         
        url = '/render?from=-10minutes&until=now&rawData=True&target=stats.gauges.crawler.redirector.s4g50.8100.nonemptychannelcount'
        crawlerUsage['nechancount'] = getGraphiteRaw(url)    
         
        url = '/render?from=-10minutes&until=now&rawData=True&target=stats.gauges.crawler.redirector.s4g50.8100.usedchannelcount'
        crawlerUsage['uchancount'] = getGraphiteRaw(url)      
        
        url = '/render?from=-10minutes&until=now&rawData=True&target=stats.gauges.crawler.downloader.s2g52.8100.totalDownloadedPageCount'
        crawlerUsage['52totaldpcount'] = getGraphiteRaw(url) 

        url = '/render?from=-10minutes&until=now&rawData=True&target=stats.gauges.crawler.downloader.s2g53.8100.totalDownloadedPageCount'
        crawlerUsage['53totaldpcount'] = getGraphiteRaw(url) 
        
        url = '/render?from=-10minutes&until=now&rawData=True&target=stats.gauges.crawler.downloader.s2g54.8100.totalDownloadedPageCount'
        crawlerUsage['54totaldpcount'] = getGraphiteRaw(url) 
        
        url = '/render?from=-10minutes&until=now&rawData=True&target=stats.gauges.crawler.downloader.s2g55.8100.totalDownloadedPageCount'
        crawlerUsage['55totaldpcount'] = getGraphiteRaw(url) 
        
        url = '/render?from=-10minutes&until=now&rawData=True&target=stats.gauges.crawler.downloader.s2g52.8100.writePageCount'
        crawlerUsage['52totalwpcount'] = getGraphiteRaw(url)
        
        url = '/render?from=-10minutes&until=now&rawData=True&target=stats.gauges.crawler.downloader.s2g53.8100.writePageCount'
        crawlerUsage['53totalwpcount'] = getGraphiteRaw(url)
        
        url = '/render?from=-10minutes&until=now&rawData=True&target=stats.gauges.crawler.downloader.s2g54.8100.writePageCount'
        crawlerUsage['54totalwpcount'] = getGraphiteRaw(url)
        
        url = '/render?from=-10minutes&until=now&rawData=True&target=stats.gauges.crawler.downloader.s2g55.8100.writePageCount'
        crawlerUsage['55totalwpcount'] = getGraphiteRaw(url)
        
    except Exception,e:
        print "catch exception in getusuage"
        print e
        print traceback.format_exc()
        return False
    return True

def sendMail():
    cmd = "cat %s | ./MailSender.py -s CrawlerDailyReport -o jingleizhang@creditease.cn -d %s -H -f render_sumwpcount.png render_sumdpcount.png render_rnechan.png render_dcachesize.png" % (reportPath, maillist)
    os.system(cmd)

def getGraphitePng():
    cmd = "rm render_*.png -v"
    os.system(cmd)
    cmd = "wget \"http://dmz.monitor/render?width=540&from=-24hours&until=now&height=360&target=sumSeries(stats.gauges.crawler.downloader.s2g*.8100.writePageCount)\" -O render_sumwpcount.png"
    os.system(cmd)
    cmd = "wget \"http://dmz.monitor/render?width=540&from=-24hours&until=now&height=360&target=stats.gauges.crawler.downloader.s2g*.8100.cachesize\" -O render_dcachesize.png"
    os.system(cmd)
    cmd = "wget \"http://dmz.monitor/render?width=540&from=-24hours&until=now&height=360&target=stats.gauges.crawler.redirector.s4g50.8100.usedchannelcount&target=stats.gauges.crawler.redirector.s4g50.8100.nonemptychannelcount\" -O render_rnechan.png"
    os.system(cmd)
    cmd = "wget \"http://dmz.monitor/render?width=540&from=-24hours&until=now&height=360&target=sumSeries(stats.gauges.crawler.downloader.s2g*.8100.totalDownloadedPageCount)\" -O render_sumdpcount.png"
    os.system(cmd)    
    
def getHtml():
    global reportPath
    reportPath = "%s/report.html" %(os.getcwd())
    f = open(reportPath,"w")
    out = "<HTML>\n"
    out += "<HEAD>"
    out += " <META charset=utf-8>"
    out += "<TITLE>Crawler Daily Report</TITLE>"
    out += "</HEAD>"
    out += "<BODY>\n"
    out += "<TABLE borderColor=#cccccc cellSpacing=0 border=1>\n";
    out += "    <TR>\n        <TD colspan=7 align='center'>Crawler Daily Report</TD>\n    </TR>\n"
    out += "    <TR bgColor=#ffeeee>\n"
    out += "        <TD>total write page count</TD>\n"
    out += "        <TD>total download page count</TD>\n"
    out += "        <TD>used chan count</TD>\n"
    out += "        <TD>no empty chan count</TD></TR>\n"
    out += "    <TR>\n"
    out += "        <TD> %s </TD>\n" % (crawlerUsage['totalwpcount'])
    out += "        <TD> %s </TD>\n" % (crawlerUsage['totaldpcount'])
    out += "        <TD> %s </TD>\n" % (crawlerUsage['uchancount'])
    out += "        <TD> %s </TD>\n" % (crawlerUsage['nechancount'])
    out += "        </TR>\n"
    out += "</TABLE><BR/><BR/><BR/>"
    f.write(out)
    
    out = "<TABLE borderColor=#cccccc cellSpacing=0 border=1>\n";
    out += "    <TR>\n<TD colspan=6 align='center'>Graphite detail info</TD>\n    </TR>\n"
    out += "    <TR bgColor=#ffeeee>\n"
    out += "        <TD>crawler ip</TD>\n"    
    out += "        <TD>crawler port</TD>\n"
    out += "        <TD>total write page</TD>\n"
    out += "        <TD>total download page</TD></TR>\n"
    out += "    <TR>\n"
    out += "        <TD>10.181.10.52</TD>\n"
    out += "        <TD>8100</TD>\n"
    out += "        <TD> %s </TD>\n" % (crawlerUsage['52totalwpcount'])
    out += "        <TD> %s </TD>\n" % (crawlerUsage['52totaldpcount'])
    out += "        </TR>\n"
    out += "    <TR>\n"
    out += "        <TD>10.181.10.53</TD>\n"
    out += "        <TD>8100</TD>\n"
    out += "        <TD> %s </TD>\n" % (crawlerUsage['53totalwpcount'])
    out += "        <TD> %s </TD>\n" % (crawlerUsage['53totaldpcount'])
    out += "        </TR>\n"    
    out += "    <TR>\n"
    out += "        <TD>10.181.10.54</TD>\n"
    out += "        <TD>8100</TD>\n"
    out += "        <TD> %s </TD>\n" % (crawlerUsage['54totalwpcount'])
    out += "        <TD> %s </TD>\n" % (crawlerUsage['54totaldpcount'])
    out += "        </TR>\n"
    out += "    <TR>\n"
    out += "        <TD>10.181.10.55</TD>\n"
    out += "        <TD>8100</TD>\n"
    out += "        <TD> %s </TD>\n" % (crawlerUsage['55totalwpcount'])
    out += "        <TD> %s </TD>\n" % (crawlerUsage['55totaldpcount'])
    out += "        </TR>\n"
    out += "</TABLE><BR/><BR/><BR/>"
    f.write(out)
    
    out = "<TABLE borderColor=#cccccc cellSpacing=0 border=1>\n";
    out += "    <TR>\n<TD colspan=6 align='center'>Graphite trend report(last 24 hours)</TD>\n    </TR>\n"
    out += "    <TR>\n"
    out += "        <TR><TD> <IMG src='cid:render_sumwpcount.png'/> </TD></TR><BR/>\n"
    out += "        <TR><TD> <IMG src='cid:render_sumdpcount.png'/> </TD></TR><BR/>\n"
    out += "        <TR><TD> <IMG src='cid:render_rnechan.png'/> </TD></TR><BR/>\n"
    out += "        <TR><TD> <IMG src='cid:render_dcachesize.png'/> </TD></TR><BR/>\n"
    out += "        </TR>\n"
    out += "</TABLE><BR/><BR/><BR/>"
    out += "</BODY>\n"
    out += "</HTML>\n"
    f.write(out)
    
    f.close()
      
    
if __name__ == '__main__':
    
    getGraphitePng()
    
    if getUsage():
        getHtml()
        sendMail()
        print "finish send crawler report."
    else:
        print "getUsage() fail!!!"
    sys.exit(0)
    
    