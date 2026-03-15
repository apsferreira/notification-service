import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { 
  BarChart3, 
  TrendingUp, 
  Mail, 
  MessageSquare,
  Calendar,
  Filter
} from 'lucide-react';

export function Analytics() {
  // Mock data for analytics
  const metrics = {
    dailyVolume: [
      { date: '2026-03-09', email: 450, sms: 120 },
      { date: '2026-03-10', email: 520, sms: 95 },
      { date: '2026-03-11', email: 380, sms: 150 },
      { date: '2026-03-12', email: 690, sms: 200 },
      { date: '2026-03-13', email: 520, sms: 180 },
      { date: '2026-03-14', email: 380, sms: 90 },
      { date: '2026-03-15', email: 320, sms: 110 },
    ],
    topTemplates: [
      { name: 'Welcome Email', sent: 1250, success_rate: 99.2 },
      { name: 'Order Confirmation', sent: 980, success_rate: 98.8 },
      { name: 'Password Reset', sent: 750, success_rate: 97.5 },
      { name: 'Payment Receipt', sent: 650, success_rate: 99.1 },
      { name: 'Shipping Update', sent: 540, success_rate: 98.0 },
    ],
    failureReasons: [
      { reason: 'Invalid email address', count: 45, percentage: 35.2 },
      { reason: 'Recipient blocked', count: 32, percentage: 25.0 },
      { reason: 'Server timeout', count: 28, percentage: 21.9 },
      { reason: 'Rate limit exceeded', count: 15, percentage: 11.7 },
      { reason: 'Template rendering error', count: 8, percentage: 6.2 },
    ]
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Analytics</h1>
          <p className="mt-2 text-gray-600">
            Detailed insights and performance metrics
          </p>
        </div>
        <div className="flex space-x-3">
          <Button variant="outline" size="sm">
            <Calendar className="mr-2 h-4 w-4" />
            Last 7 days
          </Button>
          <Button variant="outline" size="sm">
            <Filter className="mr-2 h-4 w-4" />
            Filter
          </Button>
        </div>
      </div>

      {/* Volume Chart Placeholder */}
      <Card>
        <CardHeader>
          <CardTitle>Daily Notification Volume</CardTitle>
          <CardDescription>
            Email and SMS notifications sent over the past week
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="h-64 flex items-center justify-center bg-gray-50 rounded border-2 border-dashed border-gray-300">
            <div className="text-center">
              <BarChart3 className="h-12 w-12 text-gray-400 mx-auto mb-4" />
              <p className="text-sm text-gray-500">Chart component would go here</p>
              <p className="text-xs text-gray-400 mt-2">
                Integration with Chart.js or Recharts recommended
              </p>
            </div>
          </div>
          {/* Simple data display for wireframe */}
          <div className="mt-4 grid grid-cols-7 gap-2 text-xs">
            {metrics.dailyVolume.map((day, index) => (
              <div key={index} className="text-center">
                <div className="text-gray-600 mb-1">
                  {new Date(day.date).toLocaleDateString('pt-BR', { weekday: 'short' })}
                </div>
                <div className="space-y-1">
                  <div className="bg-blue-100 text-blue-800 px-2 py-1 rounded text-xs">
                    📧 {day.email}
                  </div>
                  <div className="bg-green-100 text-green-800 px-2 py-1 rounded text-xs">
                    💬 {day.sms}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Top Templates */}
        <Card>
          <CardHeader>
            <CardTitle>Top Performing Templates</CardTitle>
            <CardDescription>
              Most used templates and their success rates
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {metrics.topTemplates.map((template, index) => (
                <div key={index} className="flex items-center justify-between p-3 border rounded">
                  <div>
                    <p className="font-medium text-sm">{template.name}</p>
                    <p className="text-xs text-gray-500">{template.sent.toLocaleString()} sent</p>
                  </div>
                  <div className="text-right">
                    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                      template.success_rate >= 99 
                        ? 'bg-green-100 text-green-800'
                        : template.success_rate >= 98
                        ? 'bg-yellow-100 text-yellow-800'
                        : 'bg-red-100 text-red-800'
                    }`}>
                      {template.success_rate}%
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Failure Analysis */}
        <Card>
          <CardHeader>
            <CardTitle>Failure Analysis</CardTitle>
            <CardDescription>
              Common reasons for notification failures
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {metrics.failureReasons.map((failure, index) => (
                <div key={index} className="flex items-center justify-between">
                  <div className="flex-1">
                    <p className="text-sm font-medium">{failure.reason}</p>
                    <div className="mt-1 bg-gray-200 rounded-full h-2">
                      <div 
                        className="bg-red-500 h-2 rounded-full"
                        style={{ width: `${failure.percentage}%` }}
                      />
                    </div>
                  </div>
                  <div className="ml-4 text-right">
                    <p className="text-sm font-medium">{failure.count}</p>
                    <p className="text-xs text-gray-500">{failure.percentage}%</p>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Performance Metrics */}
      <Card>
        <CardHeader>
          <CardTitle>Performance Metrics</CardTitle>
          <CardDescription>
            Key performance indicators for the notification service
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="text-center p-4 border rounded">
              <Mail className="h-8 w-8 text-blue-600 mx-auto mb-2" />
              <p className="text-2xl font-bold">98.2%</p>
              <p className="text-sm text-gray-600">Email Success Rate</p>
            </div>
            <div className="text-center p-4 border rounded">
              <MessageSquare className="h-8 w-8 text-green-600 mx-auto mb-2" />
              <p className="text-2xl font-bold">96.8%</p>
              <p className="text-sm text-gray-600">SMS Success Rate</p>
            </div>
            <div className="text-center p-4 border rounded">
              <TrendingUp className="h-8 w-8 text-purple-600 mx-auto mb-2" />
              <p className="text-2xl font-bold">2.3s</p>
              <p className="text-sm text-gray-600">Avg Response Time</p>
            </div>
            <div className="text-center p-4 border rounded">
              <BarChart3 className="h-8 w-8 text-orange-600 mx-auto mb-2" />
              <p className="text-2xl font-bold">99.9%</p>
              <p className="text-sm text-gray-600">Service Uptime</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}