#import <Cocoa/Cocoa.h>
#import <math.h>

extern void goDeguTick(void);
extern void goDeguKeyDown(void);
extern void goDeguSetSceneWidth(int width);
extern void goDeguSetSpeed(int speed);
extern void goDeguSetPetCount(int count);
extern void goDeguSetWheelEnabled(int enabled);
extern void goDeguSetMode(int mode);
extern void goDeguSetCoatMode(int mode);
extern void goDeguSetVariant(int variant);
extern void goDeguSetSelectedCoat(int index, int variant);
extern void goDeguSetNameLabels(int enabled);
extern void goDeguSetPetName(int index, char *value);
extern int goDeguClick(int x, int y);
extern int goDeguPetAt(int x, int y);
extern int goDeguGetSpeed(void);
extern int goDeguGetPetCount(void);
extern int goDeguGetWheelEnabled(void);
extern int goDeguGetMode(void);
extern int goDeguGetCoatMode(void);
extern int goDeguGetVariant(void);
extern int goDeguGetSelectedCoat(int index);
extern int goDeguGetVariantCount(void);
extern int goDeguGetNameLabels(void);
extern int goDeguCopyPetName(int index, char *buffer, int length);
extern int goDeguGetPetDrawX(int index);
extern int goDeguGetPetDrawY(int index);

enum {
	DeguMenuSettings = 1001,
	DeguMenuSpeedSlow = 1101,
	DeguMenuSpeedNormal = 1103,
	DeguMenuSpeedFast = 1105,
	DeguMenuCountBase = 1200,
	DeguMenuWheelEnabled = 1301,
};

static NSString *DeguVariantLabels[] = {
	@"野生色 / アグーチ",
	@"ブラック",
	@"ブルー（青みグレー）",
	@"グレー",
	@"ホワイト / クリーム",
	@"サンド / シャンパン",
	@"チョコレート",
	@"ブラックパイド",
	@"アグーチパイド",
	@"ブルーパイド（青みグレー）",
	@"クリームパイド",
};

static const NSInteger DeguMaxPetCount = 10;
static const NSInteger DeguVariantLabelCount = sizeof(DeguVariantLabels) / sizeof(DeguVariantLabels[0]);
static const CGFloat DeguSpriteWidth = 96.0;

@interface DeguView : NSView
@property(nonatomic, retain) NSImage *image;
@property(nonatomic) NSInteger hoverPet;
@end

@implementation DeguView
- (instancetype)initWithFrame:(NSRect)frame {
	self = [super initWithFrame:frame];
	if (self) {
		_hoverPet = -1;
	}
	return self;
}

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
	if (goDeguGetNameLabels() == 0 || self.hoverPet < 0 || self.hoverPet >= DeguMaxPetCount) {
		return;
	}
	char buffer[256] = {0};
	int copied = goDeguCopyPetName((int)self.hoverPet, buffer, (int)sizeof(buffer));
	NSString *name = nil;
	if (copied > 0) {
		name = [NSString stringWithUTF8String:buffer];
	}
	if (name == nil || [name length] == 0) {
		name = [NSString stringWithFormat:@"デグー%ld", (long)self.hoverPet + 1];
	}

	NSMutableParagraphStyle *style = [[[NSMutableParagraphStyle alloc] init] autorelease];
	[style setAlignment:NSTextAlignmentCenter];
	[style setLineBreakMode:NSLineBreakByTruncatingTail];
	NSDictionary *attrs = @{
		NSFontAttributeName: [NSFont systemFontOfSize:11.0 weight:NSFontWeightSemibold],
		NSForegroundColorAttributeName: [NSColor colorWithCalibratedWhite:1.0 alpha:0.96],
		NSParagraphStyleAttributeName: style
	};
	NSSize textSize = [name boundingRectWithSize:NSMakeSize(200.0, 20.0)
	                                    options:NSStringDrawingUsesLineFragmentOrigin
	                                 attributes:attrs].size;
	CGFloat labelW = MIN(MAX(72.0, ceil(textSize.width) + 22.0), 220.0);
	CGFloat labelH = 24.0;
	CGFloat petX = (CGFloat)goDeguGetPetDrawX((int)self.hoverPet);
	CGFloat petY = (CGFloat)goDeguGetPetDrawY((int)self.hoverPet);
	CGFloat x = MIN(MAX(2.0, petX + DeguSpriteWidth / 2.0 - labelW / 2.0), MAX(2.0, self.bounds.size.width - labelW - 2.0));
	CGFloat y = MAX(0.0, petY - labelH - 4.0);
	NSRect labelRect = NSMakeRect(x, y, labelW, labelH);
	NSBezierPath *path = [NSBezierPath bezierPathWithRoundedRect:labelRect xRadius:9.0 yRadius:9.0];
	[[NSColor colorWithCalibratedRed:0.13 green:0.18 blue:0.15 alpha:0.78] setFill];
	[path fill];
	[name drawInRect:NSInsetRect(labelRect, 10.0, 4.0) withAttributes:attrs];
}
@end

@interface DeguAppDelegate : NSObject <NSApplicationDelegate, NSMenuDelegate, NSTextFieldDelegate>
@property(nonatomic) CGFloat sceneHeight;
@property(nonatomic, retain) NSWindow *window;
@property(nonatomic, retain) DeguView *view;
@property(nonatomic, retain) NSStatusItem *statusItem;
@property(nonatomic, retain) NSImage *statusIcon;
@property(nonatomic, retain) NSTimer *timer;
@property(nonatomic, retain) id globalMonitor;
@property(nonatomic, retain) id localMonitor;
@property(nonatomic, retain) id mouseClickMonitor;
@property(nonatomic, retain) id mouseMoveMonitor;
@property(nonatomic, retain) NSWindow *settingsWindow;
@property(nonatomic, retain) NSPopUpButton *countPopup;
@property(nonatomic, retain) NSPopUpButton *modePopup;
@property(nonatomic, retain) NSPopUpButton *speedPopup;
@property(nonatomic, retain) NSPopUpButton *coatModePopup;
@property(nonatomic, retain) NSPopUpButton *fixedCoatPopup;
@property(nonatomic, retain) NSMutableArray *selectedCoatPopups;
@property(nonatomic, retain) NSMutableArray *petNameFields;
@property(nonatomic, retain) NSButton *wheelCheckbox;
@property(nonatomic, retain) NSButton *nameLabelsCheckbox;
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
	[menu setDelegate:self];
	NSMenuItem *title = [[[NSMenuItem alloc] initWithTitle:@"Degu Desktop" action:nil keyEquivalent:@""] autorelease];
	[title setEnabled:NO];
	[menu addItem:title];
	[menu addItem:[NSMenuItem separatorItem]];

	NSMenuItem *settings = [self menuItemWithTitle:@"設定を開く..." action:@selector(showSettings:) tag:DeguMenuSettings];
	[menu addItem:settings];
	[menu addItem:[NSMenuItem separatorItem]];

	NSMenu *speedMenu = [[[NSMenu alloc] initWithTitle:@"速さ"] autorelease];
	[speedMenu setDelegate:self];
	[speedMenu addItem:[self menuItemWithTitle:@"ゆっくり" action:@selector(setSpeed:) tag:DeguMenuSpeedSlow]];
	[speedMenu addItem:[self menuItemWithTitle:@"ふつう" action:@selector(setSpeed:) tag:DeguMenuSpeedNormal]];
	[speedMenu addItem:[self menuItemWithTitle:@"はやい" action:@selector(setSpeed:) tag:DeguMenuSpeedFast]];
	NSMenuItem *speedRoot = [[[NSMenuItem alloc] initWithTitle:@"速さ" action:nil keyEquivalent:@""] autorelease];
	[speedRoot setSubmenu:speedMenu];
	[menu addItem:speedRoot];

	NSMenu *countMenu = [[[NSMenu alloc] initWithTitle:@"表示数"] autorelease];
	[countMenu setDelegate:self];
	for (NSInteger i = 1; i <= DeguMaxPetCount; i++) {
		[countMenu addItem:[self menuItemWithTitle:[NSString stringWithFormat:@"%ld匹", (long)i] action:@selector(setPetCount:) tag:DeguMenuCountBase + i]];
	}
	NSMenuItem *countRoot = [[[NSMenuItem alloc] initWithTitle:@"表示数" action:nil keyEquivalent:@""] autorelease];
	[countRoot setSubmenu:countMenu];
	[menu addItem:countRoot];

	NSMenuItem *wheel = [self menuItemWithTitle:@"キーボード反応" action:@selector(toggleWheelEnabled:) tag:DeguMenuWheelEnabled];
	[menu addItem:wheel];
	[menu addItem:[NSMenuItem separatorItem]];

	NSMenuItem *quit = [[[NSMenuItem alloc] initWithTitle:@"終了" action:@selector(quit:) keyEquivalent:@"q"] autorelease];
	[quit setTarget:self];
	[menu addItem:quit];
	self.statusItem.menu = menu;
	[self refreshMenuState];

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
	self.mouseClickMonitor = [NSEvent addGlobalMonitorForEventsMatchingMask:NSEventMaskLeftMouseDown
	                                                                handler:^(NSEvent *event) {
		[self handleGlobalMouseDown:event];
	}];
	self.mouseMoveMonitor = [NSEvent addGlobalMonitorForEventsMatchingMask:NSEventMaskMouseMoved
	                                                               handler:^(NSEvent *event) {
		[self updateHoverFromMouseLocation:[NSEvent mouseLocation]];
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
	if (self.mouseClickMonitor != nil) {
		[NSEvent removeMonitor:self.mouseClickMonitor];
		self.mouseClickMonitor = nil;
	}
	if (self.mouseMoveMonitor != nil) {
		[NSEvent removeMonitor:self.mouseMoveMonitor];
		self.mouseMoveMonitor = nil;
	}
	[self.timer invalidate];
}

- (void)tick:(NSTimer *)timer {
	goDeguTick();
}

- (NSMenuItem *)menuItemWithTitle:(NSString *)title action:(SEL)action tag:(NSInteger)tag {
	NSMenuItem *item = [[[NSMenuItem alloc] initWithTitle:title action:action keyEquivalent:@""] autorelease];
	[item setTarget:self];
	[item setTag:tag];
	return item;
}

- (void)menuNeedsUpdate:(NSMenu *)menu {
	[self refreshMenuState];
}

- (void)menuWillOpen:(NSMenu *)menu {
	[self refreshMenuState];
}

- (void)refreshMenuState {
	if (self.statusItem == nil || self.statusItem.menu == nil) {
		return;
	}
	[self refreshMenuState:self.statusItem.menu speed:goDeguGetSpeed() count:goDeguGetPetCount() wheelEnabled:goDeguGetWheelEnabled()];
	[self refreshSettingsControls];
}

- (void)refreshMenuState:(NSMenu *)menu speed:(int)speed count:(int)count wheelEnabled:(int)wheelEnabled {
	for (NSMenuItem *item in [menu itemArray]) {
		NSInteger tag = [item tag];
		if (tag == DeguMenuSpeedSlow || tag == DeguMenuSpeedNormal || tag == DeguMenuSpeedFast) {
			int itemSpeed = 3;
			if (tag == DeguMenuSpeedSlow) {
				itemSpeed = 2;
			} else if (tag == DeguMenuSpeedFast) {
				itemSpeed = 5;
			}
			[item setState:(itemSpeed == speed) ? NSControlStateValueOn : NSControlStateValueOff];
		} else if (tag > DeguMenuCountBase && tag <= DeguMenuCountBase + DeguMaxPetCount) {
			int itemCount = (int)(tag - DeguMenuCountBase);
			[item setState:(itemCount == count) ? NSControlStateValueOn : NSControlStateValueOff];
		} else if (tag == DeguMenuWheelEnabled) {
			[item setState:wheelEnabled ? NSControlStateValueOn : NSControlStateValueOff];
		}
		if ([item submenu] != nil) {
			[self refreshMenuState:[item submenu] speed:speed count:count wheelEnabled:wheelEnabled];
		}
	}
}

- (void)setSpeed:(id)sender {
	NSInteger tag = [sender tag];
	if (tag == DeguMenuSpeedSlow) {
		goDeguSetSpeed(2);
	} else if (tag == DeguMenuSpeedFast) {
		goDeguSetSpeed(5);
	} else {
		goDeguSetSpeed(3);
	}
	[self refreshMenuState];
}

- (void)setPetCount:(id)sender {
	NSInteger tag = [sender tag];
	if (tag > DeguMenuCountBase && tag <= DeguMenuCountBase + DeguMaxPetCount) {
		goDeguSetPetCount((int)(tag - DeguMenuCountBase));
	}
	[self refreshMenuState];
}

- (void)toggleWheelEnabled:(id)sender {
	goDeguSetWheelEnabled(goDeguGetWheelEnabled() ? 0 : 1);
	[self refreshMenuState];
}

- (void)showSettings:(id)sender {
	[self ensureSettingsWindow];
	[self refreshSettingsControls];
	[self.settingsWindow makeKeyAndOrderFront:nil];
	[NSApp activateIgnoringOtherApps:YES];
}

- (NSTextField *)labelWithTitle:(NSString *)title frame:(NSRect)frame {
	NSTextField *label = [[[NSTextField alloc] initWithFrame:frame] autorelease];
	[label setStringValue:title];
	[label setBezeled:NO];
	[label setDrawsBackground:NO];
	[label setEditable:NO];
	[label setSelectable:NO];
	[label setFont:[NSFont systemFontOfSize:12.0]];
	return label;
}

- (NSPopUpButton *)popupWithFrame:(NSRect)frame action:(SEL)action {
	NSPopUpButton *popup = [[[NSPopUpButton alloc] initWithFrame:frame pullsDown:NO] autorelease];
	[popup setTarget:self];
	[popup setAction:action];
	return popup;
}

- (void)handleGlobalMouseDown:(NSEvent *)event {
	(void)event;
	NSPoint point = [NSEvent mouseLocation];
	int localX = 0;
	int localY = 0;
	if (![self localScenePointFromScreenPoint:point x:&localX y:&localY]) {
		return;
	}
	goDeguClick(localX, localY);
	[self updateHoverFromSceneX:localX y:localY];
}

- (BOOL)localScenePointFromScreenPoint:(NSPoint)point x:(int *)x y:(int *)y {
	if (self.window == nil) {
		return NO;
	}
	NSRect frame = [self.window frame];
	if (!NSPointInRect(point, frame)) {
		return NO;
	}
	CGFloat localX = point.x - frame.origin.x;
	CGFloat localY = self.sceneHeight - (point.y - frame.origin.y);
	if (localX < 0.0 || localY < 0.0 || localX >= frame.size.width || localY >= self.sceneHeight) {
		return NO;
	}
	if (x != NULL) {
		*x = (int)floor(localX);
	}
	if (y != NULL) {
		*y = (int)floor(localY);
	}
	return YES;
}

- (void)updateHoverFromMouseLocation:(NSPoint)point {
	int localX = 0;
	int localY = 0;
	if (![self localScenePointFromScreenPoint:point x:&localX y:&localY]) {
		[self updateHoverPet:-1];
		return;
	}
	[self updateHoverFromSceneX:localX y:localY];
}

- (void)updateHoverFromSceneX:(int)localX y:(int)localY {
	if (goDeguGetNameLabels() == 0) {
		[self updateHoverPet:-1];
		return;
	}
	[self updateHoverPet:goDeguPetAt(localX, localY)];
}

- (void)updateHoverPet:(NSInteger)index {
	if (self.view == nil || self.view.hoverPet == index) {
		return;
	}
	self.view.hoverPet = index;
	[self.view setNeedsDisplay:YES];
}

- (void)ensureSettingsWindow {
	if (self.settingsWindow != nil) {
		return;
	}
	NSRect frame = NSMakeRect(0, 0, 620, 560);
	self.settingsWindow = [[[NSWindow alloc] initWithContentRect:frame
	                                                   styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable | NSWindowStyleMaskMiniaturizable
	                                                     backing:NSBackingStoreBuffered
	                                                       defer:NO] autorelease];
	[self.settingsWindow setTitle:@"Degu Desktop 設定"];
	[self.settingsWindow setReleasedWhenClosed:NO];
	[self.settingsWindow center];

	NSView *content = [[[NSView alloc] initWithFrame:frame] autorelease];
	[content setAutoresizingMask:NSViewWidthSizable | NSViewHeightSizable];
	[self.settingsWindow setContentView:content];

	NSTextField *title = [self labelWithTitle:@"Degu Desktop 設定" frame:NSMakeRect(24, 516, 360, 24)];
	[title setFont:[NSFont boldSystemFontOfSize:18.0]];
	[content addSubview:title];

	NSTextField *support = [self labelWithTitle:@"対応OS: macOS 12 Monterey 以降 / Intel・Apple Silicon" frame:NSMakeRect(24, 492, 480, 20)];
	[support setTextColor:[NSColor secondaryLabelColor]];
	[content addSubview:support];

	NSTabView *tabs = [[[NSTabView alloc] initWithFrame:NSMakeRect(20, 58, 580, 420)] autorelease];
	[content addSubview:tabs];

	NSTabViewItem *animals = [[[NSTabViewItem alloc] initWithIdentifier:@"animals"] autorelease];
	[animals setLabel:@"動物"];
	NSView *animalView = [[[NSView alloc] initWithFrame:NSMakeRect(0, 0, 580, 392)] autorelease];
	[animals setView:animalView];
	[tabs addTabViewItem:animals];

	NSTabViewItem *motion = [[[NSTabViewItem alloc] initWithIdentifier:@"motion"] autorelease];
	[motion setLabel:@"動き"];
	NSView *motionView = [[[NSView alloc] initWithFrame:NSMakeRect(0, 0, 580, 392)] autorelease];
	[motion setView:motionView];
	[tabs addTabViewItem:motion];

	NSTabViewItem *names = [[[NSTabViewItem alloc] initWithIdentifier:@"names"] autorelease];
	[names setLabel:@"名前"];
	NSView *namesView = [[[NSView alloc] initWithFrame:NSMakeRect(0, 0, 580, 392)] autorelease];
	[names setView:namesView];
	[tabs addTabViewItem:names];

	[animalView addSubview:[self labelWithTitle:@"出現数" frame:NSMakeRect(22, 340, 120, 24)]];
	self.countPopup = [self popupWithFrame:NSMakeRect(150, 336, 180, 28) action:@selector(settingsCountChanged:)];
	for (NSInteger i = 1; i <= DeguMaxPetCount; i++) {
		[self.countPopup addItemWithTitle:[NSString stringWithFormat:@"%ld匹", (long)i]];
	}
	[animalView addSubview:self.countPopup];

	[animalView addSubview:[self labelWithTitle:@"色の決め方" frame:NSMakeRect(22, 298, 120, 24)]];
	self.coatModePopup = [self popupWithFrame:NSMakeRect(150, 294, 180, 28) action:@selector(settingsCoatModeChanged:)];
	for (NSString *label in @[@"固定", @"1匹ずつ選ぶ", @"ランダム"]) {
		[self.coatModePopup addItemWithTitle:label];
	}
	[animalView addSubview:self.coatModePopup];

	[animalView addSubview:[self labelWithTitle:@"決まった毛色" frame:NSMakeRect(22, 256, 120, 24)]];
	self.fixedCoatPopup = [self popupWithFrame:NSMakeRect(150, 252, 260, 28) action:@selector(settingsFixedCoatChanged:)];
	[self populateVariantPopup:self.fixedCoatPopup];
	[animalView addSubview:self.fixedCoatPopup];

	NSTextField *perPet = [self labelWithTitle:@"1匹ごとの毛色" frame:NSMakeRect(22, 214, 160, 24)];
	[perPet setFont:[NSFont boldSystemFontOfSize:12.0]];
	[animalView addSubview:perPet];

	self.selectedCoatPopups = [NSMutableArray arrayWithCapacity:DeguMaxPetCount];
	for (NSInteger i = 0; i < DeguMaxPetCount; i++) {
		NSInteger column = i / 5;
		NSInteger row = i % 5;
		CGFloat x = 22 + column * 270;
		CGFloat y = 174 - row * 34;
		[animalView addSubview:[self labelWithTitle:[NSString stringWithFormat:@"%ld匹目", (long)i + 1] frame:NSMakeRect(x, y + 4, 58, 22)]];
		NSPopUpButton *popup = [self popupWithFrame:NSMakeRect(x + 62, y, 190, 28) action:@selector(settingsSelectedCoatChanged:)];
		[popup setTag:i];
		[self populateVariantPopup:popup];
		[animalView addSubview:popup];
		[self.selectedCoatPopups addObject:popup];
	}

	[motionView addSubview:[self labelWithTitle:@"モード" frame:NSMakeRect(22, 340, 120, 24)]];
	self.modePopup = [self popupWithFrame:NSMakeRect(150, 336, 220, 28) action:@selector(settingsModeChanged:)];
	for (NSString *label in @[@"キーボード反応", @"ランダム散歩"]) {
		[self.modePopup addItemWithTitle:label];
	}
	[motionView addSubview:self.modePopup];

	[motionView addSubview:[self labelWithTitle:@"速度" frame:NSMakeRect(22, 298, 120, 24)]];
	self.speedPopup = [self popupWithFrame:NSMakeRect(150, 294, 180, 28) action:@selector(settingsSpeedChanged:)];
	for (NSString *label in @[@"ゆっくり", @"ふつう", @"はやい"]) {
		[self.speedPopup addItemWithTitle:label];
	}
	[motionView addSubview:self.speedPopup];

	self.wheelCheckbox = [[[NSButton alloc] initWithFrame:NSMakeRect(150, 248, 260, 28)] autorelease];
	[self.wheelCheckbox setButtonType:NSButtonTypeSwitch];
	[self.wheelCheckbox setTitle:@"入力中だけ回し車"];
	[self.wheelCheckbox setTarget:self];
	[self.wheelCheckbox setAction:@selector(settingsWheelChanged:)];
	[motionView addSubview:self.wheelCheckbox];

	self.nameLabelsCheckbox = [[[NSButton alloc] initWithFrame:NSMakeRect(22, 340, 240, 28)] autorelease];
	[self.nameLabelsCheckbox setButtonType:NSButtonTypeSwitch];
	[self.nameLabelsCheckbox setTitle:@"名前を表示"];
	[self.nameLabelsCheckbox setTarget:self];
	[self.nameLabelsCheckbox setAction:@selector(settingsNameLabelsChanged:)];
	[namesView addSubview:self.nameLabelsCheckbox];

	NSTextField *nameHint = [self labelWithTitle:@"ONのとき、デグーにカーソルを乗せると名前が表示されます。" frame:NSMakeRect(22, 312, 480, 22)];
	[nameHint setTextColor:[NSColor secondaryLabelColor]];
	[namesView addSubview:nameHint];

	self.petNameFields = [NSMutableArray arrayWithCapacity:DeguMaxPetCount];
	for (NSInteger i = 0; i < DeguMaxPetCount; i++) {
		NSInteger column = i / 5;
		NSInteger row = i % 5;
		CGFloat x = 22 + column * 270;
		CGFloat y = 262 - row * 42;
		[namesView addSubview:[self labelWithTitle:[NSString stringWithFormat:@"%ld匹目", (long)i + 1] frame:NSMakeRect(x, y + 4, 58, 24)]];
		NSTextField *field = [[[NSTextField alloc] initWithFrame:NSMakeRect(x + 62, y, 190, 28)] autorelease];
		[field setTag:i];
		[field setTarget:self];
		[field setAction:@selector(settingsPetNameChanged:)];
		[field setDelegate:self];
		[field setPlaceholderString:[NSString stringWithFormat:@"デグー%ld", (long)i + 1]];
		[namesView addSubview:field];
		[self.petNameFields addObject:field];
	}

	NSTextField *note = [self labelWithTitle:@"Mac版はメニューバー常駐です。Dockアイコンは通常表示しません。" frame:NSMakeRect(24, 28, 440, 20)];
	[note setTextColor:[NSColor secondaryLabelColor]];
	[content addSubview:note];

	NSButton *close = [[[NSButton alloc] initWithFrame:NSMakeRect(492, 20, 92, 32)] autorelease];
	[close setTitle:@"閉じる"];
	[close setBezelStyle:NSBezelStyleRounded];
	[close setTarget:self];
	[close setAction:@selector(closeSettings:)];
	[content addSubview:close];
}

- (void)populateVariantPopup:(NSPopUpButton *)popup {
	[popup removeAllItems];
	NSInteger count = MIN((NSInteger)goDeguGetVariantCount(), DeguVariantLabelCount);
	for (NSInteger i = 0; i < count; i++) {
		[popup addItemWithTitle:DeguVariantLabels[i]];
	}
}

- (void)refreshSettingsControls {
	if (self.settingsWindow == nil) {
		return;
	}
	NSInteger count = goDeguGetPetCount();
	[self.countPopup selectItemAtIndex:MAX(0, MIN(DeguMaxPetCount - 1, count - 1))];
	[self.coatModePopup selectItemAtIndex:goDeguGetCoatMode()];
	[self.fixedCoatPopup selectItemAtIndex:goDeguGetVariant()];
	[self.modePopup selectItemAtIndex:goDeguGetMode()];
	NSInteger speed = goDeguGetSpeed();
	[self.speedPopup selectItemAtIndex:(speed == 2 ? 0 : (speed == 5 ? 2 : 1))];
	[self.wheelCheckbox setState:goDeguGetWheelEnabled() ? NSControlStateValueOn : NSControlStateValueOff];
	for (NSInteger i = 0; i < [self.selectedCoatPopups count]; i++) {
		NSPopUpButton *popup = [self.selectedCoatPopups objectAtIndex:i];
		[popup selectItemAtIndex:goDeguGetSelectedCoat((int)i)];
		[popup setEnabled:(goDeguGetCoatMode() == 1 && i < count)];
	}
	[self.fixedCoatPopup setEnabled:(goDeguGetCoatMode() == 0)];
	[self.nameLabelsCheckbox setState:goDeguGetNameLabels() ? NSControlStateValueOn : NSControlStateValueOff];
	for (NSInteger i = 0; i < [self.petNameFields count]; i++) {
		NSTextField *field = [self.petNameFields objectAtIndex:i];
		char buffer[256] = {0};
		if (goDeguCopyPetName((int)i, buffer, (int)sizeof(buffer)) > 0) {
			NSString *name = [NSString stringWithUTF8String:buffer];
			[field setStringValue:(name != nil ? name : @"")];
		} else {
			[field setStringValue:@""];
		}
		[field setEnabled:(goDeguGetNameLabels() != 0 && i < count)];
	}
}

- (void)settingsCountChanged:(id)sender {
	NSInteger index = [sender indexOfSelectedItem];
	goDeguSetPetCount((int)index + 1);
	[self refreshMenuState];
}

- (void)settingsCoatModeChanged:(id)sender {
	goDeguSetCoatMode((int)[sender indexOfSelectedItem]);
	[self refreshMenuState];
}

- (void)settingsFixedCoatChanged:(id)sender {
	goDeguSetVariant((int)[sender indexOfSelectedItem]);
	[self refreshMenuState];
}

- (void)settingsSelectedCoatChanged:(id)sender {
	goDeguSetSelectedCoat((int)[sender tag], (int)[sender indexOfSelectedItem]);
	[self refreshMenuState];
}

- (void)settingsModeChanged:(id)sender {
	goDeguSetMode((int)[sender indexOfSelectedItem]);
	[self refreshMenuState];
}

- (void)settingsSpeedChanged:(id)sender {
	NSInteger index = [sender indexOfSelectedItem];
	int values[] = {2, 3, 5};
	goDeguSetSpeed(values[index]);
	[self refreshMenuState];
}

- (void)settingsWheelChanged:(id)sender {
	goDeguSetWheelEnabled([sender state] == NSControlStateValueOn ? 1 : 0);
	[self refreshMenuState];
}

- (void)settingsNameLabelsChanged:(id)sender {
	goDeguSetNameLabels([sender state] == NSControlStateValueOn ? 1 : 0);
	if (goDeguGetNameLabels() == 0) {
		[self updateHoverPet:-1];
	} else {
		[self updateHoverFromMouseLocation:[NSEvent mouseLocation]];
	}
	[self refreshMenuState];
}

- (void)settingsPetNameChanged:(id)sender {
	goDeguSetPetName((int)[sender tag], (char *)[[sender stringValue] UTF8String]);
	[self.view setNeedsDisplay:YES];
	[self refreshMenuState];
}

- (void)controlTextDidEndEditing:(NSNotification *)notification {
	id object = [notification object];
	if ([object isKindOfClass:[NSTextField class]] && [self.petNameFields containsObject:object]) {
		[self settingsPetNameChanged:object];
	}
}

- (void)closeSettings:(id)sender {
	[self.settingsWindow orderOut:nil];
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
