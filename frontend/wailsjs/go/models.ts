export namespace audio {

	export class AudioSource {
	    id: string;
	    label: string;
	    kind: string;
	    pid: number;
	    icon?: string;

	    static createFrom(source: any = {}) {
	        return new AudioSource(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.kind = source["kind"];
	        this.pid = source["pid"];
	        this.icon = source["icon"];
	    }
	}
	export class AudioStatus {
	    state: string;
	    source: AudioSource;
	    session: number;
	    level: number;
	    frames: number;
	    error: string;

	    static createFrom(source: any = {}) {
	        return new AudioStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.source = this.convertValues(source["source"], AudioSource);
	        this.session = source["session"];
	        this.level = source["level"];
	        this.frames = source["frames"];
	        this.error = source["error"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace screenshot {

	export class ScreenshotRect {
	    x: number;
	    y: number;
	    width: number;
	    height: number;

	    static createFrom(source: any = {}) {
	        return new ScreenshotRect(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	        this.width = source["width"];
	        this.height = source["height"];
	    }
	}
	export class ScreenshotFrame {
	    image: string;
	    width: number;
	    height: number;
	    added: boolean;
	    workArea?: ScreenshotRect;

	    static createFrom(source: any = {}) {
	        return new ScreenshotFrame(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.image = source["image"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.added = source["added"];
	        this.workArea = this.convertValues(source["workArea"], ScreenshotRect);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace subtitle {

	export class DesktopBackground {
	    image: string;
	    color: string;

	    static createFrom(source: any = {}) {
	        return new DesktopBackground(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.image = source["image"];
	        this.color = source["color"];
	    }
	}
	export class SubtitleFrame {
	    key: string;
	    source: string;
	    translation: string;
	    final: boolean;
	    incomplete: boolean;
	    pendingSeconds: number;

	    static createFrom(source: any = {}) {
	        return new SubtitleFrame(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.source = source["source"];
	        this.translation = source["translation"];
	        this.final = source["final"];
	        this.incomplete = source["incomplete"];
	        this.pendingSeconds = source["pendingSeconds"];
	    }
	}
	export class SubtitleSettings {
	    source: string;
	    target: string;
	    fontSize: number;
	    position: string;
	    customX: number;
	    customY: number;
	    style: string;

	    static createFrom(source: any = {}) {
	        return new SubtitleSettings(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.target = source["target"];
	        this.fontSize = source["fontSize"];
	        this.position = source["position"];
	        this.customX = source["customX"];
	        this.customY = source["customY"];
	        this.style = source["style"];
	    }
	}

}

