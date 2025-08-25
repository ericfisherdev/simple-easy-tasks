# Protected Files Documentation

## Overview

All file attachments in the Simple Easy Tasks application are now protected, requiring authentication and authorization to access. This prevents unauthorized access to sensitive documents and ensures data privacy.

## Security Configuration

### File Upload Restrictions

Attachments in both `tasks` and `comments` collections are configured with:

1. **Protected Access**: Files are stored with `protected: true`, requiring authentication
2. **MIME Type Allowlist**: Only the following file types are permitted:
   - Images: `image/jpeg`, `image/png`, `image/webp`, `image/gif`
   - Documents: `application/pdf`, `text/plain`, `text/markdown`
   - Data: `application/json`
3. **Size Limits**: Maximum 5MB per file (5242880 bytes)
4. **Quantity Limits**: Maximum 10 attachments per record

### Collections Affected

- **tasks.attachments**: Task-related files and documents
- **comments.attachments**: Files attached to comments

## Accessing Protected Files

### Using PocketBase's File Token System

Protected files require a signed token for access. The token includes:
- Record reference
- File identifier
- Expiration time (default: 1 hour)
- Digital signature

### API Endpoints

#### Get Protected File URL
```go
// Generate a signed URL for a protected file
fileService := services.NewFileService(pbApp)
url, err := fileService.GetProtectedFileURL("tasks", recordID, filename)
```

#### Direct File Access
```
GET /api/files/{collection}/{recordId}/{filename}?token={signed_token}
```

### Access Control Rules

#### Task Attachments
Users can access task attachments if they are:
- The task assignee
- The task creator
- A member of the task's project

#### Comment Attachments
Users can access comment attachments if they are:
- The comment author
- Have access to the associated task (see task rules above)

## Implementation Examples

### Frontend (JavaScript)
```javascript
// Fetch protected file URL from backend
async function getProtectedFileURL(collection, recordId, filename) {
    const response = await fetch(`/api/files/url`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${authToken}`
        },
        body: JSON.stringify({
            collection,
            recordId,
            filename
        })
    });
    
    const data = await response.json();
    return data.url;
}

// Display protected image
async function displayProtectedImage(imgElement, collection, recordId, filename) {
    const url = await getProtectedFileURL(collection, recordId, filename);
    imgElement.src = url;
}
```

### Backend (Go)
```go
// Handler for generating protected file URLs
func (h *FileHandler) GetProtectedURL(c *gin.Context) {
    var req struct {
        Collection string `json:"collection" binding:"required"`
        RecordID   string `json:"recordId" binding:"required"`
        Filename   string `json:"filename" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Validate user access
    userID := c.GetString("user_id")
    hasAccess, err := h.fileService.ValidateFileAccess(
        userID, 
        req.Collection, 
        req.RecordID,
    )
    
    if err != nil || !hasAccess {
        c.JSON(403, gin.H{"error": "Access denied"})
        return
    }
    
    // Generate signed URL
    url, err := h.fileService.GetProtectedFileURL(
        req.Collection, 
        req.RecordID, 
        req.Filename,
    )
    
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to generate URL"})
        return
    }
    
    c.JSON(200, gin.H{"url": url})
}
```

## Migration Guide

### For Existing Files

If migrating from unprotected to protected files:

1. **Update Collection Schema**: Apply the new schema with `protected: true`
2. **Run Migration**: Execute PocketBase migration to update existing records
3. **Update Frontend**: Replace direct file URLs with token-based URLs
4. **Test Access**: Verify all file access paths work correctly

### Database Migration
```bash
# Apply the updated schema
make pocketbase-migrate

# Or manually
go run cmd/pocketbase/main.go migrate up
```

## Security Best Practices

1. **Never expose direct file paths**: Always use the token-based URL system
2. **Implement proper access control**: Verify user permissions before generating URLs
3. **Use short token expiration**: Default 1 hour is recommended
4. **Validate MIME types**: Ensure uploaded files match the allowlist
5. **Monitor file access**: Log file access attempts for security auditing
6. **Regular security reviews**: Periodically review file access patterns

## Troubleshooting

### Common Issues

1. **403 Forbidden Error**: User lacks permission to access the file
2. **Token Expired**: Generate a new token (tokens expire after 1 hour)
3. **Invalid MIME Type**: File type not in the allowlist
4. **File Too Large**: File exceeds 5MB limit

### Debug Checklist

- [ ] User is authenticated
- [ ] User has permission to access the record
- [ ] File token is valid and not expired
- [ ] File exists in the collection
- [ ] MIME type is allowed

## API Reference

See `internal/services/file_service.go` for the complete implementation of the file service with protected access support.