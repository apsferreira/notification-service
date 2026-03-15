import { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { 
  Search, 
  Filter, 
  Mail, 
  MessageSquare,
  RefreshCw,
  AlertCircle,
  CheckCircle,
  Clock,
  XCircle,
  Eye,
  Download
} from 'lucide-react';
import { getStatusColor, formatDate } from '@/lib/utils';

export function Notifications() {
  const [selectedStatus, setSelectedStatus] = useState<'all' | 'sent' | 'pending' | 'failed' | 'retrying'>('all');
  const [selectedType, setSelectedType] = useState<'all' | 'email' | 'sms'>('all');

  // Mock notifications data
  const notifications = [
    {
      id: '1',
      type: 'email',
      recipient: 'user@example.com',
      subject: 'Welcome to Instituto Itinerante, João!',
      body: 'Hello João, welcome to our platform!',
      template_id: 'template-1',
      template_name: 'Welcome Email',
      variables: { customer_name: 'João' },
      status: 'sent',
      attempts: 1,
      created_at: '2026-03-15T08:30:00Z',
      sent_at: '2026-03-15T08:30:15Z',
      error: null
    },
    {
      id: '2',
      type: 'sms',
      recipient: '+55 11 99999-9999',
      subject: '',
      body: 'Your verification code is 123456. Valid for 5 minutes.',
      template_id: 'template-3',
      template_name: 'SMS Verification',
      variables: { code: '123456' },
      status: 'failed',
      attempts: 3,
      created_at: '2026-03-15T08:25:00Z',
      sent_at: null,
      error: 'Invalid phone number format'
    },
    {
      id: '3',
      type: 'email',
      recipient: 'customer@company.com',
      subject: 'Order #12345 confirmed',
      body: 'Thank you for your order! Order #12345 has been confirmed.',
      template_id: 'template-2',
      template_name: 'Order Confirmation',
      variables: { order_id: '12345' },
      status: 'pending',
      attempts: 0,
      created_at: '2026-03-15T08:20:00Z',
      sent_at: null,
      error: null
    },
    {
      id: '4',
      type: 'email',
      recipient: 'admin@test.com',
      subject: 'Reset your password',
      body: 'Click here to reset your password.',
      template_id: 'template-4',
      template_name: 'Password Reset',
      variables: { reset_link: 'https://example.com/reset/token123' },
      status: 'retrying',
      attempts: 2,
      created_at: '2026-03-15T08:15:00Z',
      sent_at: null,
      error: 'Server timeout'
    }
  ];

  const filteredNotifications = notifications.filter(n => {
    if (selectedStatus !== 'all' && n.status !== selectedStatus) return false;
    if (selectedType !== 'all' && n.type !== selectedType) return false;
    return true;
  });

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'sent':
        return <CheckCircle className="h-4 w-4 text-green-600" />;
      case 'failed':
        return <XCircle className="h-4 w-4 text-red-600" />;
      case 'pending':
        return <Clock className="h-4 w-4 text-yellow-600" />;
      case 'retrying':
        return <RefreshCw className="h-4 w-4 text-orange-600" />;
      default:
        return <AlertCircle className="h-4 w-4 text-gray-600" />;
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Notifications</h1>
          <p className="mt-2 text-gray-600">
            Audit log and notification history
          </p>
        </div>
        <div className="flex space-x-3">
          <Button variant="outline">
            <Download className="mr-2 h-4 w-4" />
            Export
          </Button>
          <Button variant="outline">
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
        </div>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="pt-6">
          <div className="space-y-4">
            <div className="flex space-x-4 items-center">
              <div className="flex-1 relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-4 w-4" />
                <input
                  type="text"
                  placeholder="Search by recipient, template, or content..."
                  className="pl-10 pr-4 py-2 w-full border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                />
              </div>
              <Button variant="outline" size="sm">
                <Filter className="mr-2 h-4 w-4" />
                Advanced Filter
              </Button>
            </div>

            <div className="flex flex-wrap gap-2">
              {/* Status Filter */}
              <div className="flex space-x-2">
                <span className="text-sm font-medium text-gray-700 py-2">Status:</span>
                {['all', 'sent', 'pending', 'failed', 'retrying'].map((status) => (
                  <Button
                    key={status}
                    variant={selectedStatus === status ? 'default' : 'outline'}
                    size="sm"
                    onClick={() => setSelectedStatus(status as any)}
                  >
                    {status === 'all' ? 'All' : status.charAt(0).toUpperCase() + status.slice(1)}
                  </Button>
                ))}
              </div>

              {/* Type Filter */}
              <div className="flex space-x-2">
                <span className="text-sm font-medium text-gray-700 py-2">Type:</span>
                {['all', 'email', 'sms'].map((type) => (
                  <Button
                    key={type}
                    variant={selectedType === type ? 'default' : 'outline'}
                    size="sm"
                    onClick={() => setSelectedType(type as any)}
                  >
                    {type === 'all' ? (
                      'All Types'
                    ) : type === 'email' ? (
                      <>
                        <Mail className="mr-1 h-4 w-4" />
                        Email
                      </>
                    ) : (
                      <>
                        <MessageSquare className="mr-1 h-4 w-4" />
                        SMS
                      </>
                    )}
                  </Button>
                ))}
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Notifications List */}
      <Card>
        <CardHeader>
          <CardTitle>Notification History</CardTitle>
          <CardDescription>
            {filteredNotifications.length} notifications found
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {filteredNotifications.map((notification) => (
              <div 
                key={notification.id} 
                className="flex items-start justify-between p-4 border rounded-lg hover:bg-gray-50 transition-colors"
              >
                <div className="flex items-start space-x-4 flex-1">
                  <div className="flex-shrink-0 mt-1">
                    {notification.type === 'email' ? (
                      <Mail className="h-5 w-5 text-blue-600" />
                    ) : (
                      <MessageSquare className="h-5 w-5 text-green-600" />
                    )}
                  </div>
                  
                  <div className="flex-1">
                    <div className="flex items-center space-x-2 mb-2">
                      <p className="font-medium text-gray-900">
                        {notification.template_name}
                      </p>
                      {getStatusIcon(notification.status)}
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(notification.status)}`}>
                        {notification.status}
                      </span>
                    </div>
                    
                    <p className="text-sm text-gray-600 mb-1">
                      To: <span className="font-mono">{notification.recipient}</span>
                    </p>
                    
                    {notification.subject && (
                      <p className="text-sm text-gray-600 mb-1">
                        Subject: {notification.subject}
                      </p>
                    )}
                    
                    <p className="text-sm text-gray-500 line-clamp-2">
                      {notification.body}
                    </p>
                    
                    {notification.error && (
                      <p className="text-sm text-red-600 mt-2">
                        <AlertCircle className="inline h-4 w-4 mr-1" />
                        Error: {notification.error}
                      </p>
                    )}
                  </div>
                </div>

                <div className="flex items-center space-x-4 text-sm text-gray-500">
                  <div className="text-right">
                    <p>Attempts: {notification.attempts}</p>
                    <p>{formatDate(notification.created_at)}</p>
                    {notification.sent_at && (
                      <p className="text-green-600">
                        Sent: {formatDate(notification.sent_at)}
                      </p>
                    )}
                  </div>
                  
                  <Button variant="ghost" size="sm">
                    <Eye className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>

          {/* Pagination Placeholder */}
          <div className="mt-6 flex items-center justify-between">
            <p className="text-sm text-gray-600">
              Showing 1-{filteredNotifications.length} of {notifications.length} notifications
            </p>
            <div className="flex space-x-2">
              <Button variant="outline" size="sm" disabled>Previous</Button>
              <Button variant="outline" size="sm" disabled>Next</Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Notification Details Modal Wireframe */}
      <Card className="border-blue-200 bg-blue-50">
        <CardHeader>
          <CardTitle className="text-blue-800">Notification Details Modal (Wireframe)</CardTitle>
          <CardDescription>
            Detailed view when clicking on a notification
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
            <div className="space-y-2">
              <h4 className="font-medium">Basic Info:</h4>
              <ul className="space-y-1 text-gray-600">
                <li>• Full notification content</li>
                <li>• Template variables used</li>
                <li>• Delivery timeline</li>
                <li>• Retry attempts history</li>
              </ul>
            </div>
            <div className="space-y-2">
              <h4 className="font-medium">Actions:</h4>
              <ul className="space-y-1 text-gray-600">
                <li>• Retry failed notifications</li>
                <li>• View raw template data</li>
                <li>• Export notification details</li>
                <li>• Copy for debugging</li>
              </ul>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}