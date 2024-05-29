<% if (ScanStarted) { %>
### 🔄 **Security scan started**

#### Scan details
<%= ScanDetails %>
<% } %>

<% if (ScanPassed) { %>
### ✅ Security scan passed

#### Scan details
<%= ScanDetails %>
<% } %>

<% if (ScanFailed) { %>
### ❌ Security scan failed

#### Results of the scan
<%= ScanResults %>

#### Scan details
<%= ScanDetails %>
<% } %>

<% if (ScanCrashed) { %>
### 🚧 Security scan crashed

#### Error details
<%= ErrorDetails %>

#### Scan details
<%= ScanDetails %>
<% } %>