Distributed Crawler Cluster in Golang
==============

# Architecture

The service is consist of two cluster: downloader cluster and redirector cluster.

## Downloader

Downloader will do following things:

* accept links from post request, and put links in Q1
* download webpages of these links, and put pages in Q2
* extract links from downloaded webpages
* post extracted links to redirector cluster
* dump webpages to disk

## Redirector

Redirector will do following things:

* accept links from post request, and pust links in Q3
* filter links we have crawled before
* filter links we do not want to crawl (e.g. using regex)
* prioritize these links and post them to downloader cluster
* control post frequency of links from different host, e.g. we can not crawl one site too frequently

In order to control crawl frequency of different site, redirector will creat N channels, and links from host a will be always send to h(a) % N channel. Then, every channel will have a goroutine to process links in it, and every goroutine will control process speed by itself.

## Collector

Collector will do following things:

* collect downloaded webpages from disk on downloader cluster
* writer these pages to HDFS

## Analyzer

Redirector will only post important links to downloader online. By analyze downloaded webpages in HDFS, we will find more links we want to crawl.

## Information Extractor(IE)

IE will extract structed data from webpages in HDFS and save them in database, etc.

# Golang Crawler Need to Know

* 