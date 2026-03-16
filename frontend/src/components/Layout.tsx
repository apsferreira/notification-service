import type { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { 
  BarChart3, 
  Bell, 
  FileText, 
  Home,
  Settings
} from 'lucide-react';
import { cn } from '@/lib/utils';

interface LayoutProps {
  children: ReactNode;
}

const navigation = [
  { name: 'Dashboard', href: '/dashboard', icon: Home },
  { name: 'Analytics', href: '/analytics', icon: BarChart3 },
  { name: 'Templates', href: '/templates', icon: FileText },
  { name: 'Notifications', href: '/notifications', icon: Bell },
];

export function Layout({ children }: LayoutProps) {
  const location = useLocation();

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="flex h-screen">
        {/* Sidebar */}
        <div className="w-64 bg-white shadow-lg">
          <div className="flex flex-col h-full">
            {/* Logo */}
            <div className="flex items-center px-6 py-4 border-b">
              <Bell className="h-8 w-8 text-blue-600" />
              <span className="ml-2 text-xl font-semibold text-gray-900">
                Notification Admin
              </span>
            </div>

            {/* Navigation */}
            <nav className="flex-1 px-4 py-6 space-y-2">
              {navigation.map((item) => {
                const isActive = location.pathname === item.href;
                return (
                  <Link
                    key={item.name}
                    to={item.href}
                    className={cn(
                      'flex items-center px-4 py-2 text-sm font-medium rounded-md transition-colors',
                      isActive
                        ? 'bg-blue-100 text-blue-700'
                        : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100'
                    )}
                  >
                    <item.icon className="mr-3 h-5 w-5" />
                    {item.name}
                  </Link>
                );
              })}
            </nav>

            {/* Settings */}
            <div className="px-4 py-4 border-t">
              <Link
                to="/settings"
                className="flex items-center px-4 py-2 text-sm font-medium text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-md transition-colors"
              >
                <Settings className="mr-3 h-5 w-5" />
                Settings
              </Link>
            </div>
          </div>
        </div>

        {/* Main content */}
        <div className="flex-1 overflow-auto">
          <main className="p-6">
            {children}
          </main>
        </div>
      </div>
    </div>
  );
}