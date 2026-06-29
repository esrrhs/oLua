package main

import (
	"io"
	"log"
	"testing"
)

// suppressLog 在 benchmark 时关闭 log 输出（减少干扰）。
func suppressLog() func() {
	old := log.Writer()
	log.SetOutput(io.Discard)
	return func() { log.SetOutput(old) }
}

// BenchmarkTableAccessRealworld 测量在真实游戏模块代码上的优化耗时。
// 该文件包含约 100 行代码、多个方法、多种访问模式（self.physics.velocity 等）。
func BenchmarkTableAccessRealworld(b *testing.B) {
	defer suppressLog()()
	*opt_table_access_threshold = 2
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := runOptimizer("input/table_access_realworld.lua")
		if err != nil {
			b.Fatalf("optimizer failed: %v", err)
		}
	}
}

// BenchmarkTableAccessDeepChain 测量深层链 (a.b.c.d.e...) 优化的耗时。
func BenchmarkTableAccessDeepChain(b *testing.B) {
	defer suppressLog()()
	*opt_table_access_threshold = 2
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := runOptimizer("input/table_access_advanced.lua")
		if err != nil {
			b.Fatalf("optimizer failed: %v", err)
		}
	}
}

// BenchmarkTableAccessLoop 测量循环嵌套场景的优化耗时。
func BenchmarkTableAccessLoop(b *testing.B) {
	defer suppressLog()()
	*opt_table_access_threshold = 2
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := runOptimizer("input/table_access_loop.lua")
		if err != nil {
			b.Fatalf("optimizer failed: %v", err)
		}
	}
}

// BenchmarkIsPathPrefix 测量路径前缀判断函数性能（被高频调用）。
func BenchmarkIsPathPrefix(b *testing.B) {
	for i := 0; i < b.N; i++ {
		isPathPrefix("a.b", "a.b.c.d.e")
	}
}

// BenchmarkContainTableAccess 测量字符串单词边界匹配的性能。
func BenchmarkContainTableAccess(b *testing.B) {
	content := "    self_transform_position.x = self_transform_position.x + self_physics_velocity.x * dt"
	for i := 0; i < b.N; i++ {
		contain_table_access(content, "self_transform_position")
	}
}
