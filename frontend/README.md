# Notification Service Admin Dashboard

Admin dashboard for managing the Instituto Itinerante notification service, providing analytics, template management, and audit capabilities.

## Features

### 🏠 Dashboard Overview
- **Key Metrics**: Total notifications, success rate, active templates, pending notifications
- **Recent Activity**: Latest notification activity with status indicators
- **Trend Indicators**: Performance changes from previous periods

### 📊 Analytics
- **Volume Charts**: Daily email/SMS notification volume (placeholder for Chart.js/Recharts)
- **Performance Metrics**: Success rates, response times, service uptime
- **Top Templates**: Most used templates with success rates
- **Failure Analysis**: Common failure reasons with percentage breakdowns

### 📧 Template Management
- **CRUD Operations**: Create, read, update, delete notification templates
- **Template Types**: Email and SMS template support
- **Variable System**: Dynamic placeholder support (`{{variable}}`)
- **Preview**: Template preview with sample data
- **Usage Analytics**: Template usage statistics and performance

### 🔍 Notification Audit Log
- **Searchable History**: Full notification history with advanced filtering
- **Status Tracking**: Real-time status updates (pending, sent, failed, retrying)
- **Error Details**: Detailed error information for failed notifications
- **Retry Management**: Retry failed notifications
- **Export Capabilities**: Export notification data for analysis

## Tech Stack

- **Frontend**: React 19.2 + TypeScript
- **Styling**: Tailwind CSS + shadcn/ui components
- **Icons**: Lucide React
- **Routing**: React Router v6
- **Build Tool**: Vite

## Project Structure

```
notification-admin/
├── src/
│   ├── components/
│   │   ├── ui/           # shadcn/ui base components
│   │   └── Layout.tsx    # App layout with sidebar
│   ├── lib/
│   │   └── utils.ts      # Shared utilities
│   ├── pages/
│   │   ├── Dashboard.tsx # Main dashboard overview
│   │   ├── Analytics.tsx # Performance analytics
│   │   ├── Templates.tsx # Template management
│   │   └── Notifications.tsx # Audit log
│   └── App.tsx          # Main app with routing
├── public/              # Static assets
├── index.html          # HTML entry point
└── package.json        # Dependencies
```

## Data Models

### Notification
```typescript
interface Notification {
  id: string;
  type: 'email' | 'sms';
  recipient: string;
  subject: string;
  body: string;
  template_id?: string;
  variables?: Record<string, any>;
  status: 'pending' | 'sent' | 'failed' | 'retrying';
  attempts: number;
  error?: string;
  created_at: string;
  sent_at?: string;
}
```

### Template
```typescript
interface Template {
  id: string;
  name: string;
  type: 'email' | 'sms';
  subject_template: string;
  body_template: string;
  created_at: string;
  updated_at: string;
}
```

## Key Features Implemented

### Dashboard Components
- [x] Metrics cards with trend indicators
- [x] Recent activity feed with status badges
- [x] Responsive grid layout
- [x] Status icon system

### Analytics Views
- [x] Performance metrics display
- [x] Top templates ranking
- [x] Failure analysis breakdown
- [x] Chart placeholder for data visualization

### Template Management
- [x] Template grid with filtering
- [x] Type-based filtering (email/SMS)
- [x] Action buttons (view, edit, copy, delete)
- [x] Usage statistics display
- [x] Template editor wireframe

### Audit Log
- [x] Searchable notification list
- [x] Multi-criteria filtering
- [x] Status-based grouping
- [x] Error detail display
- [x] Pagination placeholder

## Wireframes & UI Patterns

### Status System
- **Sent**: Green with checkmark icon
- **Failed**: Red with X icon  
- **Pending**: Yellow with clock icon
- **Retrying**: Orange with refresh icon

### Color Coding
- **Primary**: Blue (#3b82f6) - navigation and primary actions
- **Success**: Green (#10b981) - successful operations
- **Warning**: Yellow (#f59e0b) - pending/warning states
- **Error**: Red (#ef4444) - failures and errors

### Layout Pattern
- Fixed sidebar navigation (256px)
- Main content area with 24px padding
- Card-based information display
- Consistent spacing (24px between sections)

## Development Setup

```bash
# Navigate to project
cd /projects/admin-dashboards/notification-admin

# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build
```

## API Integration Points

### Notification Service Endpoints
- `GET /notifications` - List notifications with filtering
- `GET /notifications/{id}` - Get notification details
- `POST /notifications/send` - Send new notification
- `GET /templates` - List templates
- `POST /templates` - Create template
- `PUT /templates/{id}` - Update template
- `DELETE /templates/{id}` - Delete template

### Expected Data Sources
- Real-time notification status from notification-service
- Template usage analytics
- Performance metrics and trends
- Error logs and failure analysis

## Future Enhancements

### Planned Features
- [ ] Real-time WebSocket connection for live updates
- [ ] Chart integration (Chart.js or Recharts)
- [ ] Rich text editor for email templates
- [ ] Template variable autocomplete
- [ ] Bulk operations (resend, delete)
- [ ] Advanced filtering and search
- [ ] Export functionality (CSV, PDF)
- [ ] User role management
- [ ] Template approval workflow

### Technical Improvements
- [ ] API client integration
- [ ] Loading and error states
- [ ] Form validation
- [ ] Optimistic updates
- [ ] Infinite scrolling for large datasets
- [ ] Mobile responsive enhancements

## Integration with Backend

This dashboard is designed to integrate with the notification-service backend located at `/projects/notification-service/`. The service provides:

- RESTful API endpoints for CRUD operations
- OpenTelemetry metrics for monitoring
- PostgreSQL database with audit logging
- Template rendering with variable support
- Email/SMS delivery with retry logic

The admin dashboard provides the management interface for this backend service, enabling operators to monitor performance, manage templates, and troubleshoot delivery issues.

## Deployment Considerations

- Static build can be served from any CDN
- Environment configuration for API endpoints
- Authentication integration required
- CORS configuration needed for API access
- Consider reverse proxy for API routing