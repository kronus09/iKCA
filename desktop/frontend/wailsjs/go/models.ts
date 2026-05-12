export namespace main {
	
	export class APIResponse {
	    success: boolean;
	    message?: string;
	    data?: any;
	
	    static createFrom(source: any = {}) {
	        return new APIResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.data = source["data"];
	    }
	}
	export class GenerateParams {
	    country: string;
	    org: string;
	    ca_name: string;
	    domain: string;
	    client_names: string[];
	    shared_san: string;
	    ca_lifetime: number;
	    cert_lifetime: number;
	    ca_pass: string;
	    client_pass: string;
	
	    static createFrom(source: any = {}) {
	        return new GenerateParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.country = source["country"];
	        this.org = source["org"];
	        this.ca_name = source["ca_name"];
	        this.domain = source["domain"];
	        this.client_names = source["client_names"];
	        this.shared_san = source["shared_san"];
	        this.ca_lifetime = source["ca_lifetime"];
	        this.cert_lifetime = source["cert_lifetime"];
	        this.ca_pass = source["ca_pass"];
	        this.client_pass = source["client_pass"];
	    }
	}
	export class  {
	    name: string;
	    subject: string;
	    not_after: string;
	
	    static createFrom(source: any = {}) {
	        return new (source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.subject = source["subject"];
	        this.not_after = source["not_after"];
	    }
	}
	export class SavedConfig {
	    country: string;
	    org: string;
	    ca_name: string;
	    domain: string;
	    client_names: string[];
	    shared_san: string;
	    ca_lifetime: number;
	    cert_lifetime: number;
	    ca_pass: string;
	    client_pass: string;
	    ca_subject: string;
	    ca_not_after: string;
	    server_subject: string;
	    server_not_after: string;
	    clients: [];
	    generated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new SavedConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.country = source["country"];
	        this.org = source["org"];
	        this.ca_name = source["ca_name"];
	        this.domain = source["domain"];
	        this.client_names = source["client_names"];
	        this.shared_san = source["shared_san"];
	        this.ca_lifetime = source["ca_lifetime"];
	        this.cert_lifetime = source["cert_lifetime"];
	        this.ca_pass = source["ca_pass"];
	        this.client_pass = source["client_pass"];
	        this.ca_subject = source["ca_subject"];
	        this.ca_not_after = source["ca_not_after"];
	        this.server_subject = source["server_subject"];
	        this.server_not_after = source["server_not_after"];
	        this.clients = this.convertValues(source["clients"], );
	        this.generated_at = source["generated_at"];
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

