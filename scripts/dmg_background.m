// 生成 DMG 安装窗口背景图（660x400）
// 用法: dmg_background <output.png>
// 编译: clang -framework Cocoa dmg_background.m -o dmg_background
#import <Cocoa/Cocoa.h>

#pragma clang diagnostic ignored "-Wdeprecated-declarations"

int main(int argc, const char *argv[]) {
    if (argc != 2) {
        fprintf(stderr, "usage: dmg_background <output.png>\n");
        return 1;
    }
    const int W = 660, H = 400;

    NSImage *img = [[NSImage alloc] initWithSize:NSMakeSize(W, H)];
    [img lockFocus];

    // 浅色渐变背景
    NSGradient *bg = [[NSGradient alloc]
        initWithStartingColor:[NSColor colorWithCalibratedWhite:0.985 alpha:1.0]
                  endingColor:[NSColor colorWithCalibratedWhite:0.93 alpha:1.0]];
    [bg drawInRect:NSMakeRect(0, 0, W, H) angle:270];

    // 居中段落样式
    NSMutableParagraphStyle *center = [[NSMutableParagraphStyle alloc] init];
    center.alignment = NSTextAlignmentCenter;

    // 主标题（窗口坐标 y=50 → 图片坐标 H-50-32）
    NSDictionary *title = @{
        NSFontAttributeName: [NSFont boldSystemFontOfSize:22],
        NSForegroundColorAttributeName: [NSColor colorWithCalibratedWhite:0.18 alpha:1.0],
        NSParagraphStyleAttributeName: center,
    };
    [@"拖入 Applications 文件夹完成安装" drawInRect:NSMakeRect(0, H - 82, W, 32) withAttributes:title];

    // 副标题
    NSDictionary *sub = @{
        NSFontAttributeName: [NSFont systemFontOfSize:13],
        NSForegroundColorAttributeName: [NSColor colorWithCalibratedWhite:0.45 alpha:1.0],
        NSParagraphStyleAttributeName: center,
    };
    [@"将左侧 Audio Switch 拖到右侧 Applications 文件夹" drawInRect:NSMakeRect(0, H - 112, W, 22) withAttributes:sub];

    // 箭头：连接左（app 图标）与右（Applications 图标）
    // 图标 128px：app 在窗口 {150,180}，Applications 在 {460,180}，图标中心线 y=244 → 图片坐标 y=156
    NSColor *arrow = [NSColor systemBlueColor];
    [arrow set];
    NSBezierPath *line = [NSBezierPath bezierPath];
    [line setLineWidth:9];
    [line setLineCapStyle:NSRoundLineCapStyle];
    [line moveToPoint:NSMakePoint(305, 156)];
    [line lineToPoint:NSMakePoint(420, 156)];
    [line stroke];
    NSBezierPath *head = [NSBezierPath bezierPath];
    [head moveToPoint:NSMakePoint(448, 156)];
    [head lineToPoint:NSMakePoint(414, 174)];
    [head lineToPoint:NSMakePoint(414, 138)];
    [head closePath];
    [head fill];

    [img unlockFocus];

    // 导出 PNG（新 SDK 已移除 NSImage representationUsingType，用 CGImageDestination）
    CGImageRef cg = [img CGImageForProposedRect:NULL context:nil hints:nil];
    if (cg == NULL) {
        fprintf(stderr, "dmg_background: 生成图像失败\n");
        return 1;
    }
    CFURLRef url = CFURLCreateFromFileSystemRepresentation(
        NULL, (const UInt8 *)argv[1], strlen(argv[1]), false);
    CGImageDestinationRef dest = CGImageDestinationCreateWithURL(
        url, CFSTR("public.png"), 1, NULL);
    if (dest == NULL) {
        fprintf(stderr, "dmg_background: 无法创建输出文件 %s\n", argv[1]);
        return 1;
    }
    CGImageDestinationAddImage(dest, cg, NULL);
    if (!CGImageDestinationFinalize(dest)) {
        fprintf(stderr, "dmg_background: 写入 PNG 失败\n");
        return 1;
    }
    CFRelease(dest);
    CFRelease(url);
    return 0;
}
