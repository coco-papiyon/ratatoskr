export namespace app {
	
	export class LocalEntry {
	    id: string;
	    name: string;
	    kind: string;
	    path: string;
	    modifiedAt?: string;
	    size?: number;
	    archivePath?: string;
	    archiveEntry?: string;
	
	    static createFrom(source: any = {}) {
	        return new LocalEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.path = source["path"];
	        this.modifiedAt = source["modifiedAt"];
	        this.size = source["size"];
	        this.archivePath = source["archivePath"];
	        this.archiveEntry = source["archiveEntry"];
	    }
	}
	export class S3Preview {
	    content: string;
	    dataUrl?: string;
	
	    static createFrom(source: any = {}) {
	        return new S3Preview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content = source["content"];
	        this.dataUrl = source["dataUrl"];
	    }
	}
	export class StructuredTable {
	    ruleName: string;
	    columns: string[];
	    rows: string[][];
	
	    static createFrom(source: any = {}) {
	        return new StructuredTable(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ruleName = source["ruleName"];
	        this.columns = source["columns"];
	        this.rows = source["rows"];
	    }
	}
	export class StructuredTableRule {
	    name: string;
	    filePattern: string;
	    jq: string;
	
	    static createFrom(source: any = {}) {
	        return new StructuredTableRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.filePattern = source["filePattern"];
	        this.jq = source["jq"];
	    }
	}
	export class ViewerConfig {
	    extensions: Record<string, Array<string>>;
	    proxy: string;
	    certificate: string;
	
	    static createFrom(source: any = {}) {
	        return new ViewerConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.extensions = source["extensions"];
	        this.proxy = source["proxy"];
	        this.certificate = source["certificate"];
	    }
	}

}

