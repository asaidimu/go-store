import { defineConfig } from 'vitepress';

export default defineConfig({
  title: "Documentation",
  lang: 'en-US',
  base: '/',
  lastUpdated: true,
  cleanUrls: true,

  head: [
    ['link', { rel: 'icon', href: '/favicon.ico' }]
  ],

  themeConfig: {
    nav: [
    {
        "text": "Getting Started",
        "link": "/getting-started"
    },
    {
        "text": "Core Operations",
        "link": "/core-operations"
    },
    {
        "text": "Task-Based Guide",
        "link": "/task-based-guide"
    },
    {
        "text": "Advanced Usage",
        "link": "/advanced-usage"
    },
    {
        "text": "Problem Solving",
        "link": "/problem-solving"
    },
    {
        "text": "Development & Contributing",
        "link": "/development-contributing"
    },
    {
        "text": "Project Architecture",
        "link": "/project-architecture"
    },
    {
        "text": "Reference",
        "link": "/reference/system"
    }
],
    sidebar: [
    {
        "text": "Getting Started",
        "items": [
            {
                "text": "Prerequisites",
                "link": "/getting-started#prerequisites"
            },
            {
                "text": "Installation Steps",
                "link": "/getting-started#installation-steps"
            },
            {
                "text": "Verification",
                "link": "/getting-started#verification"
            },
            {
                "text": "Basic Document Operations",
                "link": "/getting-started#basic-document-operations"
            }
        ]
    },
    {
        "text": "Core Operations",
        "items": [
            {
                "text": "Initializing the Store",
                "link": "/core-operations#initializing-the-store"
            },
            {
                "text": "Inserting Documents",
                "link": "/core-operations#inserting-documents"
            },
            {
                "text": "Retrieving Documents",
                "link": "/core-operations#retrieving-documents"
            },
            {
                "text": "Updating Documents",
                "link": "/core-operations#updating-documents"
            },
            {
                "text": "Deleting Documents",
                "link": "/core-operations#deleting-documents"
            }
        ]
    },
    {
        "text": "Task-Based Guide",
        "items": [
            {
                "text": "Indexing and Querying Data",
                "link": "/task-based-guide#indexing-and-querying-data"
            },
            {
                "text": "Streaming Documents",
                "link": "/task-based-guide#streaming-documents"
            }
        ]
    },
    {
        "text": "Advanced Usage",
        "items": [
            {
                "text": "Advanced Usage",
                "link": "/advanced-usage#advanced-usage"
            },
            {
                "text": "Concurrent Operations",
                "link": "/advanced-usage#concurrent-operations"
            },
            {
                "text": "Understanding Concurrency Control and Memory Management",
                "link": "/advanced-usage#understanding-concurrency-control-and-memory-management"
            },
            {
                "text": "Importance of `Close()` Method",
                "link": "/advanced-usage#importance-of-close-method"
            },
            {
                "text": "Data Copying Behavior",
                "link": "/advanced-usage#data-copying-behavior"
            }
        ]
    },
    {
        "text": "Problem Solving",
        "items": [
            {
                "text": "Problem Solving",
                "link": "/problem-solving#problem-solving"
            },
            {
                "text": "Troubleshooting Common Errors",
                "link": "/problem-solving#troubleshooting-common-errors"
            },
            {
                "text": "FAQ (Frequently Asked Questions)",
                "link": "/problem-solving#faq-frequently-asked-questions"
            }
        ]
    },
    {
        "text": "Development & Contributing",
        "items": [
            {
                "text": "Development & Contributing",
                "link": "/development-contributing#development-contributing"
            },
            {
                "text": "Development Setup",
                "link": "/development-contributing#development-setup"
            },
            {
                "text": "Scripts",
                "link": "/development-contributing#scripts"
            },
            {
                "text": "Testing",
                "link": "/development-contributing#testing"
            },
            {
                "text": "Contributing Guidelines",
                "link": "/development-contributing#contributing-guidelines"
            },
            {
                "text": "Issue Reporting",
                "link": "/development-contributing#issue-reporting"
            }
        ]
    },
    {
        "text": "Project Architecture",
        "items": [
            {
                "text": "Project Architecture",
                "link": "/project-architecture#project-architecture"
            },
            {
                "text": "Data Flow",
                "link": "/project-architecture#data-flow"
            }
        ]
    },
    {
        "text": "Types",
        "items": [
            {
                "text": "DocumentResult",
                "link": "/reference/types#documentresult"
            }
        ]
    },
    {
        "text": "Interfaces",
        "items": [
            {
                "text": "DocumentLike",
                "link": "/reference/interfaces#documentlike"
            },
            {
                "text": "Cursor",
                "link": "/reference/interfaces#cursor"
            },
            {
                "text": "Document",
                "link": "/reference/interfaces#document"
            },
            {
                "text": "DocumentStream",
                "link": "/reference/interfaces#documentstream"
            }
        ]
    },
    {
        "text": "Examples",
        "items": [
            {
                "text": "Basic Document CRUD",
                "link": "/reference/patterns#basic-document-crud"
            },
            {
                "text": "Indexed Lookup",
                "link": "/reference/patterns#indexed-lookup"
            },
            {
                "text": "Streaming Documents",
                "link": "/reference/patterns#streaming-documents"
            },
            {
                "text": "Concurrent Update Pattern",
                "link": "/reference/patterns#concurrent-update-pattern"
            }
        ]
    },
    {
        "text": "Reference",
        "items": [
            {
                "text": "System Overview",
                "link": "/reference/system"
            },
            {
                "text": "Dependencies",
                "link": "/reference/dependencies"
            },
            {
                "text": "Integration",
                "link": "/reference/integration"
            },
            {
                "text": "Methods",
                "link": "/reference/methods"
            },
            {
                "text": "Error Reference",
                "link": "/reference/errors"
            }
        ]
    }
],

    socialLinks: [
      // { icon: 'github', link: '[https://github.com/asaidimu/$](https://github.com/asaidimu/$){data.reference.system.name.toLowerCase()}' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2023-present go-store'
    }
  }
});