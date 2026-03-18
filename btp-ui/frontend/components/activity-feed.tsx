import { ActivityItem } from "@/lib/types";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

type ActivityFeedProps = {
  items: ActivityItem[];
};

const typeColorMap: Record<string, string> = {
  upload_request_created: "bg-blue-50 text-blue-700",
  patient_approved_upload: "bg-emerald-50 text-emerald-700",
  aes_key_generated: "bg-indigo-50 text-indigo-700",
  key_encrypted: "bg-purple-50 text-purple-700",
  access_request_created: "bg-yellow-50 text-yellow-700",
  hospital_approved_access: "bg-cyan-50 text-cyan-700",
  key_shared: "bg-indigo-50 text-indigo-700",
  access_granted: "bg-emerald-50 text-emerald-700",
  access_revoked: "bg-red-50 text-red-700",
  access_expired: "bg-orange-50 text-orange-700"
};

export function ActivityFeed({ items }: ActivityFeedProps) {
  return (
    <Card className="p-6">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-lg font-semibold text-slate-900">Blockchain Activity</h3>
        <Badge className="bg-slate-100 text-slate-600">Live</Badge>
      </div>
      <div className="space-y-3">
        {items.length === 0 ? (
          <p className="text-sm text-slate-500">No activity yet.</p>
        ) : (
          items.map((item) => (
            <div
              key={item.id}
              className="rounded-lg border border-slate-100 bg-slate-50/80 px-4 py-3"
            >
              <div className="mb-1 flex items-center justify-between gap-2">
                <Badge
                  className={typeColorMap[item.type] || "bg-slate-100 text-slate-700"}
                >
                  {item.type}
                </Badge>
                <span className="text-xs text-slate-400">
                  {new Date(item.timestamp).toLocaleTimeString()}
                </span>
              </div>
              <p className="text-sm text-slate-700">{item.message}</p>
            </div>
          ))
        )}
      </div>
    </Card>
  );
}
