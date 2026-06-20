#import <Cocoa/Cocoa.h>

extern void goDeguTick(void);
extern void goDeguKeyDown(void);
extern void goDeguSetSceneWidth(int width);

@interface DeguView : NSView
@property(nonatomic, retain) NSImage *image;
@end

@implementation DeguView
- (BOOL)isOpaque {
	return NO;
}

- (BOOL)isFlipped {
	return YES;
}

- (void)drawRect:(NSRect)dirtyRect {
	[[NSColor clearColor] setFill];
	NSRectFill(self.bounds);
	if (self.image != nil) {
		[self.image drawInRect:self.bounds];
	}
}
@end

@interface DeguAppDelegate : NSObject <NSApplicationDelegate>
@property(nonatomic) CGFloat sceneHeight;
@property(nonatomic, retain) NSWindow *window;
@property(nonatomic, retain) DeguView *view;
@property(nonatomic, retain) NSStatusItem *statusItem;
@property(nonatomic, retain) NSImage *statusIcon;
@property(nonatomic, retain) NSTimer *timer;
@property(nonatomic, retain) id globalMonitor;
@property(nonatomic, retain) id localMonitor;
- (instancetype)initWithSceneHeight:(CGFloat)sceneHeight iconBytes:(const unsigned char *)iconBytes iconLength:(int)iconLength;
@end

static DeguAppDelegate *deguDelegate = nil;

@implementation DeguAppDelegate
- (instancetype)initWithSceneHeight:(CGFloat)sceneHeight iconBytes:(const unsigned char *)iconBytes iconLength:(int)iconLength {
	self = [super init];
	if (self) {
		_sceneHeight = sceneHeight;
		if (iconBytes != NULL && iconLength > 0) {
			NSData *data = [NSData dataWithBytes:iconBytes length:(NSUInteger)iconLength];
			_statusIcon = [[NSImage alloc] initWithData:data];
			[_statusIcon setSize:NSMakeSize(20, 18)];
			[_statusIcon setTemplate:NO];
		}
	}
	return self;
}

- (void)applicationDidFinishLaunching:(NSNotification *)notification {
	NSScreen *screen = [NSScreen mainScreen];
	NSRect visible = [screen visibleFrame];
	CGFloat width = visible.size.width;
	if (width < 320.0) {
		width = 320.0;
	}

	NSRect frame = NSMakeRect(visible.origin.x, visible.origin.y, width, self.sceneHeight);
	self.window = [[[NSWindow alloc] initWithContentRect:frame
	                                           styleMask:NSWindowStyleMaskBorderless
	                                             backing:NSBackingStoreBuffered
	                                               defer:NO] autorelease];
	[self.window setReleasedWhenClosed:NO];
	[self.window setOpaque:NO];
	[self.window setBackgroundColor:[NSColor clearColor]];
	[self.window setHasShadow:NO];
	[self.window setIgnoresMouseEvents:YES];
	[self.window setCanHide:NO];
	[self.window setLevel:NSStatusWindowLevel];
	[self.window setCollectionBehavior:NSWindowCollectionBehaviorCanJoinAllSpaces |
	                                    NSWindowCollectionBehaviorStationary |
	                                    NSWindowCollectionBehaviorIgnoresCycle];

	self.view = [[[DeguView alloc] initWithFrame:NSMakeRect(0, 0, width, self.sceneHeight)] autorelease];
	[self.view setAutoresizingMask:NSViewWidthSizable | NSViewHeightSizable];
	[self.window setContentView:self.view];
	[self.window orderFrontRegardless];
	goDeguSetSceneWidth((int)width);

	self.statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
	if (self.statusIcon != nil) {
		self.statusItem.button.image = self.statusIcon;
		self.statusItem.button.imagePosition = NSImageOnly;
	} else {
		self.statusItem.button.title = @"Degu";
	}
	self.statusItem.button.toolTip = @"Degu Desktop";

	NSMenu *menu = [[[NSMenu alloc] initWithTitle:@"Degu Desktop"] autorelease];
	NSMenuItem *title = [[[NSMenuItem alloc] initWithTitle:@"Degu Desktop" action:nil keyEquivalent:@""] autorelease];
	[title setEnabled:NO];
	[menu addItem:title];
	[menu addItem:[NSMenuItem separatorItem]];
	NSMenuItem *quit = [[[NSMenuItem alloc] initWithTitle:@"Quit" action:@selector(quit:) keyEquivalent:@"q"] autorelease];
	[quit setTarget:self];
	[menu addItem:quit];
	self.statusItem.menu = menu;

	self.timer = [NSTimer timerWithTimeInterval:0.055
	                                     target:self
	                                   selector:@selector(tick:)
	                                   userInfo:nil
	                                    repeats:YES];
	[[NSRunLoop mainRunLoop] addTimer:self.timer forMode:NSRunLoopCommonModes];

	self.globalMonitor = [NSEvent addGlobalMonitorForEventsMatchingMask:NSEventMaskKeyDown
	                                                             handler:^(NSEvent *event) {
		goDeguKeyDown();
	}];
	self.localMonitor = [NSEvent addLocalMonitorForEventsMatchingMask:NSEventMaskKeyDown
	                                                          handler:^NSEvent *(NSEvent *event) {
		goDeguKeyDown();
		return event;
	}];
}

- (void)applicationWillTerminate:(NSNotification *)notification {
	if (self.globalMonitor != nil) {
		[NSEvent removeMonitor:self.globalMonitor];
		self.globalMonitor = nil;
	}
	if (self.localMonitor != nil) {
		[NSEvent removeMonitor:self.localMonitor];
		self.localMonitor = nil;
	}
	[self.timer invalidate];
}

- (void)tick:(NSTimer *)timer {
	goDeguTick();
}

- (void)quit:(id)sender {
	[NSApp terminate:nil];
}
@end

void updateDeguImage(const unsigned char *bytes, int length, int width, int height) {
	if (deguDelegate == nil || deguDelegate.view == nil || bytes == NULL || length <= 0) {
		return;
	}
	NSData *data = [NSData dataWithBytes:bytes length:(NSUInteger)length];
	NSImage *image = [[[NSImage alloc] initWithData:data] autorelease];
	if (image == nil) {
		return;
	}
	[image setSize:NSMakeSize(width, height)];
	deguDelegate.view.image = image;
	[deguDelegate.view setNeedsDisplay:YES];
}

void startDeguApp(int sceneHeight, const unsigned char *iconBytes, int iconLength) {
	@autoreleasepool {
		NSApplication *app = [NSApplication sharedApplication];
		deguDelegate = [[DeguAppDelegate alloc] initWithSceneHeight:(CGFloat)sceneHeight iconBytes:iconBytes iconLength:iconLength];
		[app setDelegate:deguDelegate];
		[app setActivationPolicy:NSApplicationActivationPolicyAccessory];
		[app run];
	}
}
