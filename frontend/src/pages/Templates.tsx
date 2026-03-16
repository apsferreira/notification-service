import { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { 
  Plus, 
  Search, 
  Mail, 
  MessageSquare,
  Edit,
  Trash2,
  Eye,
  Copy
} from 'lucide-react';
import { formatDate } from '@/lib/utils';

export function Templates() {
  const [selectedType, setSelectedType] = useState<'all' | 'email' | 'sms'>('all');

  // Mock templates data
  const templates = [
    {
      id: '1',
      name: 'Welcome Email',
      type: 'email',
      subject_template: 'Welcome to Instituto Itinerante, {{customer_name}}!',
      body_template: '<h1>Hello {{customer_name}}</h1><p>Welcome to our platform!</p>',
      created_at: '2026-03-10T10:00:00Z',
      updated_at: '2026-03-10T10:00:00Z',
      usage_count: 1250
    },
    {
      id: '2',
      name: 'Order Confirmation',
      type: 'email',
      subject_template: 'Order #{{order_id}} confirmed',
      body_template: '<h2>Thank you for your order!</h2><p>Order #{{order_id}} has been confirmed.</p>',
      created_at: '2026-03-08T15:30:00Z',
      updated_at: '2026-03-12T09:15:00Z',
      usage_count: 980
    },
    {
      id: '3',
      name: 'SMS Verification',
      type: 'sms',
      subject_template: '',
      body_template: 'Your verification code is {{code}}. Valid for 5 minutes.',
      created_at: '2026-03-05T14:20:00Z',
      updated_at: '2026-03-05T14:20:00Z',
      usage_count: 750
    },
    {
      id: '4',
      name: 'Password Reset',
      type: 'email',
      subject_template: 'Reset your password',
      body_template: '<p>Click <a href="{{reset_link}}">here</a> to reset your password.</p>',
      created_at: '2026-03-01T11:45:00Z',
      updated_at: '2026-03-01T11:45:00Z',
      usage_count: 420
    }
  ];

  const filteredTemplates = selectedType === 'all' 
    ? templates 
    : templates.filter(t => t.type === selectedType);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Templates</h1>
          <p className="mt-2 text-gray-600">
            Manage email and SMS notification templates
          </p>
        </div>
        <Button>
          <Plus className="mr-2 h-4 w-4" />
          Create Template
        </Button>
      </div>

      {/* Filters */}
      <div className="flex space-x-4 items-center">
        <div className="flex-1 relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-4 w-4" />
          <input
            type="text"
            placeholder="Search templates..."
            className="pl-10 pr-4 py-2 w-full border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          />
        </div>
        <div className="flex space-x-2">
          <Button 
            variant={selectedType === 'all' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setSelectedType('all')}
          >
            All Types
          </Button>
          <Button 
            variant={selectedType === 'email' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setSelectedType('email')}
          >
            <Mail className="mr-2 h-4 w-4" />
            Email
          </Button>
          <Button 
            variant={selectedType === 'sms' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setSelectedType('sms')}
          >
            <MessageSquare className="mr-2 h-4 w-4" />
            SMS
          </Button>
        </div>
      </div>

      {/* Templates Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {filteredTemplates.map((template) => (
          <Card key={template.id} className="hover:shadow-lg transition-shadow">
            <CardHeader>
              <div className="flex items-start justify-between">
                <div>
                  <CardTitle className="text-lg">{template.name}</CardTitle>
                  <CardDescription>
                    <span className="inline-flex items-center">
                      {template.type === 'email' ? (
                        <Mail className="mr-1 h-4 w-4" />
                      ) : (
                        <MessageSquare className="mr-1 h-4 w-4" />
                      )}
                      {template.type.toUpperCase()}
                    </span>
                  </CardDescription>
                </div>
                <div className="flex space-x-1">
                  <Button variant="ghost" size="sm">
                    <Eye className="h-4 w-4" />
                  </Button>
                  <Button variant="ghost" size="sm">
                    <Copy className="h-4 w-4" />
                  </Button>
                  <Button variant="ghost" size="sm">
                    <Edit className="h-4 w-4" />
                  </Button>
                  <Button variant="ghost" size="sm">
                    <Trash2 className="h-4 w-4 text-red-500" />
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {template.subject_template && (
                  <div>
                    <p className="text-sm font-medium text-gray-700">Subject:</p>
                    <p className="text-sm text-gray-600 truncate">
                      {template.subject_template}
                    </p>
                  </div>
                )}
                <div>
                  <p className="text-sm font-medium text-gray-700">Body Preview:</p>
                  <p className="text-sm text-gray-600 line-clamp-3">
                    {template.body_template.replace(/<[^>]*>/g, '')}
                  </p>
                </div>
                <div className="flex justify-between items-center text-sm text-gray-500">
                  <span>{template.usage_count} sent</span>
                  <span>{formatDate(template.updated_at)}</span>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Create Template Wireframe */}
      <Card className="border-dashed border-2">
        <CardContent className="pt-6">
          <div className="text-center py-8">
            <Plus className="h-12 w-12 text-gray-400 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">
              Create New Template
            </h3>
            <p className="text-gray-600 mb-6">
              Design email and SMS templates with dynamic variables
            </p>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              Start Creating
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Template Editor Modal Wireframe */}
      <Card className="border-blue-200 bg-blue-50">
        <CardHeader>
          <CardTitle className="text-blue-800">Template Editor (Modal Wireframe)</CardTitle>
          <CardDescription>
            Features for the template creation/editing modal
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
              <div className="space-y-2">
                <h4 className="font-medium">Form Fields:</h4>
                <ul className="space-y-1 text-gray-600">
                  <li>• Template Name (required)</li>
                  <li>• Type: Email/SMS (required)</li>
                  <li>• Subject (email only)</li>
                  <li>• Body Content (rich text editor)</li>
                  <li>• Variable placeholders preview</li>
                </ul>
              </div>
              <div className="space-y-2">
                <h4 className="font-medium">Features:</h4>
                <ul className="space-y-1 text-gray-600">
                  <li>• Variable insertion helper</li>
                  <li>• Live preview with sample data</li>
                  <li>• HTML editor for email templates</li>
                  <li>• Character counter for SMS</li>
                  <li>• Template validation</li>
                </ul>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}