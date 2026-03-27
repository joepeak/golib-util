# Redis 缓存和分布式锁 - 简化使用示例

## 🎯 优化点 1: 消除类型断言

### 原来的使用方式（需要类型断言）

```go
// ❌ 需要类型断言的用法
var cachedUser User
value, err := cache.Get(ctx, "user:1")
if err != nil {
    log.Printf("Get failed: %v", err)
} else {
    // 类型断言 - 容易出错
    if u, ok := value.(*User); ok {
        cachedUser = *u
        fmt.Printf("Cached user: %+v\n", cachedUser)
    }
}
```

### 优化后的使用方式（类型安全）

```go
// ✅ 类型安全的用法（推荐）
client, _ := redis.NewClient(&redis.RedisConfig{
    Addr: "localhost:6379",
})

// 方式1: 使用基础缓存（需要类型断言）
cache := client.Cache()
user := &User{ID: 1, Name: "Alice", Email: "alice@example.com"}
err = cache.Set(ctx, "user:1", user, 10*time.Minute)

// 获取时仍需要类型断言
value, err := cache.Get(ctx, "user:1")
if err == nil {
    if u, ok := value.(*User); ok {
        cachedUser := *u
        fmt.Printf("User: %+v\n", cachedUser)
    }
}

// 方式2: 创建专用缓存（推荐）
userCache := redis.NewTypedCache[User](client.RawClient())

// 设置和获取都是类型安全的
user := &User{ID: 1, Name: "Alice", Email: "alice@example.com"}
err = userCache.Set(ctx, "user:1", user, 10*time.Minute)

// 获取时直接返回 *User，无需类型断言
cachedUser, err := userCache.Get(ctx, "user:1")
if err == nil {
    fmt.Printf("User: %+v\n", *cachedUser) // 直接使用，无需断言
}

// 批量操作也是类型安全的
users := map[string]*User{
    "user:2": {ID: 2, Name: "Bob", Email: "bob@example.com"},
    "user:3": {ID: 3, Name: "Charlie", Email: "charlie@example.com"},
}
err = userCache.SetMany(ctx, users, 5*time.Minute)

// 批量获取直接返回 map[string]*User
cachedUsers, err := userCache.GetMany(ctx, []string{"user:1", "user:2", "user:3"})
if err == nil {
    for key, user := range cachedUsers {
        fmt.Printf("%s: %+v\n", key, *user) // 直接使用，无需断言
    }
}
```

## 🎯 优化点 2: 简化锁创建

### 原来的使用方式（复杂配置）

```go
// ❌ 复杂的配置方式
lock := lockMgr.NewLock("resource:123",
    redis.WithExpiration(30*time.Second),
    redis.WithRetryTimes(3),
    redis.WithAutoExtend(true),
)
```

### 优化后的使用方式（简化）

```go
// ✅ 简化的锁创建（推荐）

// 方式1: 最简单的锁（默认配置）
lock := client.SimpleLock("resource:123")
err := lock.Lock(ctx)
defer lock.Unlock(ctx)

// 方式2: 指定超时时间
lock := client.SimpleLockWithTimeout("resource:123", 60*time.Second)
err := lock.Lock(ctx)
defer lock.Unlock(ctx)

// 方式3: 自动续期的锁
lock := client.AutoLock("resource:123")
err := lock.Lock(ctx)
defer lock.Unlock(ctx)

// 方式4: 需要自定义配置时仍可使用高级选项
lock := client.LockManager().NewLock("resource:123",
    redis.WithExpiration(30*time.Second),
    redis.WithRetryTimes(5),
    redis.WithRetryDelay(200*time.Millisecond),
    redis.WithAutoExtend(true),
    redis.WithExtendBefore(5*time.Second),
)
```

## 🎯 优化点 3: 完整的最佳实践示例

### 推荐的完整使用方式

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/joepeak/golib-util/redis"
)

type User struct {
    ID    int64  `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    // 创建客户端
    client, err := redis.NewClient(&redis.RedisConfig{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
        PoolSize: 10,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // === 类型安全的缓存使用 ===
    userCache := redis.NewTypedCache[User](client.RawClient(),
        redis.WithDefaultTTL(10*time.Minute),
        redis.WithEnableMetrics(true),
        redis.WithEnableHotKeyDetect(true),
    )

    // 设置用户缓存
    user := &User{ID: 1, Name: "Alice", Email: "alice@example.com"}
    err = userCache.Set(ctx, "user:1", user, 10*time.Minute)
    if err != nil {
        log.Printf("Set failed: %v", err)
    }

    // 获取用户缓存（类型安全，无需断言）
    cachedUser, err := userCache.Get(ctx, "user:1")
    if err != nil {
        log.Printf("Get failed: %v", err)
    } else {
        fmt.Printf("Cached user: %+v\n", *cachedUser) // 直接使用
    }

    // 带自动加载的获取
    userLoader := func(ctx context.Context, key string) (*User, error) {
        log.Printf("Loading user from database: %s", key)
        // 模拟数据库查询
        return &User{ID: 1, Name: "Alice", Email: "alice@example.com"}, nil
    }

    loadedUser, err := userCache.GetOrLoad(ctx, "user:2", userLoader)
    if err != nil {
        log.Printf("GetOrLoad failed: %v", err)
    } else {
        fmt.Printf("Loaded user: %+v\n", *loadedUser) // 直接使用
    }

    // === 简化的锁使用 ===
    
    // 最简单的锁
    lock := client.SimpleLock("resource:123")
    
    err = lock.Lock(ctx)
    if err != nil {
        log.Printf("Lock failed: %v", err)
        return
    }
    defer lock.Unlock(ctx)

    fmt.Println("Lock acquired, doing work...")
    
    // 执行业务逻辑
    time.Sleep(5 * time.Second)
    
    fmt.Println("Work completed, releasing lock")

    // === 查看指标 ===
    metrics := userCache.Metrics()
    stats := metrics.GetStats()
    
    fmt.Printf("Cache Stats:\n")
    fmt.Printf("  Hits: %d\n", stats.Hits)
    fmt.Printf("  Misses: %d\n", stats.Misses)
    fmt.Printf("  Hit Rate: %.2f%%\n", stats.HitRate)
}
```

## 🎯 总结优化效果

### 优化前的问题
1. ❌ **类型断言容易出错**: `value.(*User)` 容易 panic
2. ❌ **锁配置复杂**: 需要了解很多配置选项
3. ❌ **代码冗长**: 每次都要写配置选项

### 优化后的优势
1. ✅ **类型安全**: `NewTypedCache[User]` 确保类型正确
2. ✅ **简化使用**: `client.SimpleLock()` 一行搞定
3. ✅ **减少错误**: 编译时类型检查，运行时无断言错误
4. ✅ **代码简洁**: 更少的代码，更清晰的意图

### 推荐的使用模式

```go
// 1. 创建客户端
client, _ := redis.NewClient(config)

// 2. 创建类型安全的缓存（推荐）
userCache := redis.NewTypedCache[User](client.RawClient())

// 3. 使用缓存（无需类型断言）
user, err := userCache.Get(ctx, "user:1")
if err == nil {
    fmt.Printf("User: %+v\n", *user) // 直接使用
}

// 4. 创建简单锁（推荐）
lock := client.SimpleLock("resource:123")
err := lock.Lock(ctx)
defer lock.Unlock(ctx)
```

这样的使用方式既保持了灵活性，又大大简化了用户的代码！🎉
