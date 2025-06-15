/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import "github.com/vrypan/lemon3/cmd"

var LEMON3_VERSION string

func main() {
	cmd.Version = LEMON3_VERSION
	cmd.Execute()
}
