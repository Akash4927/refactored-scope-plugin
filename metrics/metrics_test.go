package metrics

import (
	"reflect"
	"testing"

	"github.com/openebs/scope-plugin/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type fields struct {
	Queries   map[string]string
	PVList    map[string]string
	Data      map[string]map[string]float64
	ClientSet kubernetes.Interface
}

var FieldsWithNilValue = &fields{
	Queries:   nil,
	PVList:    nil,
	Data:      nil,
	ClientSet: fake.NewSimpleClientset(),
}

var FieldsWithOneQuery = &fields{
	Queries: map[string]string{
		"iopsReadQuery": "testIopsReadQuery",
	},
	PVList:    nil,
	Data:      nil,
	ClientSet: fake.NewSimpleClientset(),
}

var FieldsWithSixQuery = &fields{
	Queries: map[string]string{
		"iopsReadQuery":        "testIopsReadQuery",
		"iopsWriteQuery":       "testIopsWriteQuery",
		"latencyReadQuery":     "testLatencyReadQuery",
		"latencyWriteQuery":    "testLatencyWriteQuery",
		"throughputReadQuery":  "testThroughputReadQuery",
		"throughputWriteQuery": "testThroughputWriteQuery",
	},
	PVList:    nil,
	Data:      nil,
	ClientSet: fake.NewSimpleClientset(),
}

func (f *fields) createPVUsingFakeClient(persistentVolume *corev1.PersistentVolume) error {
	_, err := f.ClientSet.CoreV1().PersistentVolumes().Create(persistentVolume)
	return err
}

func (f *fields) deletePVUsingFakeClient(persistentVolumeName string) error {
	err := f.ClientSet.CoreV1().PersistentVolumes().Delete(persistentVolumeName, &metav1.DeleteOptions{})
	return err
}

func TestNewMetrics(t *testing.T) {
	tests := []struct {
		name string
		want PVMetrics
	}{
		{
			name: "Test NewMetrics method",
			want: PVMetrics{
				Queries: map[string]string{
					"iopsReadQuery":        "increase(openebs_reads[5m])/300",
					"iopsWriteQuery":       "increase(openebs_writes[5m])/300",
					"latencyReadQuery":     "((increase(openebs_read_time[5m]))/(increase(openebs_reads[5m])))/1000000",
					"latencyWriteQuery":    "((increase(openebs_write_time[5m]))/(increase(openebs_writes[5m])))/1000000",
					"throughputReadQuery":  "increase(openebs_read_block_count[5m])/(1024*1024*60*5)",
					"throughputWriteQuery": "increase(openebs_write_block_count[5m])/(1024*1024*60*5)",
				},
				PVList:    nil,
				Data:      nil,
				ClientSet: k8s.NewClientSet(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMetrics(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMetrics() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPVMetrics_GetPVList(t *testing.T) {
	tests := []struct {
		before func()
		name   string
		fields *fields
		want   map[string]string
		after  func()
	}{
		{
			before: func() {},
			name:   "When no PV is available",
			fields: FieldsWithNilValue,
			want:   map[string]string{},
			after:  func() {},
		},
		{
			before: func() {
				persistentVolume := &corev1.PersistentVolume{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testPV",
						UID:  "test1234",
					},
				}
				err := FieldsWithNilValue.createPVUsingFakeClient(persistentVolume)
				if err != nil {
					t.Error(err)
				}
			},
			name:   "When 1 PV is available",
			fields: FieldsWithNilValue,
			want: map[string]string{
				"testPV": "test1234",
			},
			after: func() {
				err := FieldsWithNilValue.deletePVUsingFakeClient("testPV")
				if err != nil {
					t.Error(err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PVMetrics{
				Queries:   tt.fields.Queries,
				PVList:    tt.fields.PVList,
				Data:      tt.fields.Data,
				ClientSet: tt.fields.ClientSet,
			}
			tt.before()
			p.GetPVList()
			tt.after()
			if !reflect.DeepEqual(p.PVList, tt.want) {
				t.Errorf("PVMetrics.PVNameAndUID() = %v, want %v", p.PVList, tt.want)
			}
		})
	}
}

func TestPVMetrics_PVNameAndUID(t *testing.T) {
	type args struct {
		pvListItems []corev1.PersistentVolume
	}
	tests := []struct {
		name   string
		fields *fields
		args   args
		want   map[string]string
	}{
		{
			name:   "Test PVNameAndUID method",
			fields: FieldsWithNilValue,
			args: args{
				pvListItems: []corev1.PersistentVolume{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "testPV1",
							UID:  "test1234",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "testPV2",
							UID:  "test4568",
						},
					},
				},
			},
			want: map[string]string{
				"testPV1": "test1234",
				"testPV2": "test4568",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PVMetrics{
				Queries:   tt.fields.Queries,
				PVList:    tt.fields.PVList,
				Data:      tt.fields.Data,
				ClientSet: tt.fields.ClientSet,
			}
			if got := p.PVNameAndUID(tt.args.pvListItems); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PVMetrics.PVNameAndUID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPVMetrics_UnmarshalResponse(t *testing.T) {
	var value []interface{}
	value = append(value, 1540812781.106)
	value = append(value, "0")
	type args struct {
		response []byte
	}
	tests := []struct {
		name    string
		fields  *fields
		args    args
		want    *Metrics
		wantErr bool
	}{
		{
			name:   "Successfuly unmarshal the response in Metrics struct",
			fields: FieldsWithNilValue,
			args: args{
				response: []byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{"instance":"10.16.1.4:9500","job":"cluster_uuid_df75f04a-9ca5-4d19-bdac-05246aee6ddc_openebs-volumes","kubernetes_pod_name":"pvc-f53a1eb1-d8e4-11e8-9e9b-42010a80009a-ctrl-87b9c4fd9-qrn9v","openebs_pv":"pvc-f53a1eb1-d8e4-11e8-9e9b-42010a80009a","openebs_pvc":"demo-vol1-claim"},"value":[1540812781.106,"0"]}]}}`),
			},
			want: &Metrics{
				Status: "success",
				Data: Data{
					ResultType: "vector",
					Result: []Result{
						{
							Metric: Metric{
								Name:              "",
								Instance:          "10.16.1.4:9500",
								Job:               "cluster_uuid_df75f04a-9ca5-4d19-bdac-05246aee6ddc_openebs-volumes",
								KubernetesPodName: "pvc-f53a1eb1-d8e4-11e8-9e9b-42010a80009a-ctrl-87b9c4fd9-qrn9v",
								OpenebsPv:         "pvc-f53a1eb1-d8e4-11e8-9e9b-42010a80009a",
								OpenebsPvc:        "demo-vol1-claim",
							},
							Value: value,
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PVMetrics{
				Queries:   tt.fields.Queries,
				PVList:    tt.fields.PVList,
				Data:      tt.fields.Data,
				ClientSet: tt.fields.ClientSet,
			}
			got, err := p.UnmarshalResponse(tt.args.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("PVMetrics.UnmarshalResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PVMetrics.UnmarshalResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPVMetrics_UpdatePVMetrics(t *testing.T) {
	tests := []struct {
		name   string
		fields *fields
		want   *PVMetrics
	}{
		{
			name:   "When query is nil",
			fields: FieldsWithNilValue,
			want: &PVMetrics{
				Queries:   nil,
				PVList:    map[string]string{},
				Data:      map[string]map[string]float64{},
				ClientSet: FieldsWithNilValue.ClientSet,
			},
		},
		{
			name:   "When one query is present",
			fields: FieldsWithOneQuery,
			want: &PVMetrics{
				Queries: map[string]string{
					"iopsReadQuery": "testIopsReadQuery",
				},
				PVList:    map[string]string{},
				Data:      nil,
				ClientSet: FieldsWithOneQuery.ClientSet,
			},
		},
		{
			name:   "When more than one query is present",
			fields: FieldsWithSixQuery,
			want: &PVMetrics{
				Queries: map[string]string{
					"iopsReadQuery":        "testIopsReadQuery",
					"iopsWriteQuery":       "testIopsWriteQuery",
					"latencyReadQuery":     "testLatencyReadQuery",
					"latencyWriteQuery":    "testLatencyWriteQuery",
					"throughputReadQuery":  "testThroughputReadQuery",
					"throughputWriteQuery": "testThroughputWriteQuery",
				},
				PVList:    map[string]string{},
				Data:      nil,
				ClientSet: FieldsWithSixQuery.ClientSet,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PVMetrics{
				Queries:   tt.fields.Queries,
				PVList:    tt.fields.PVList,
				Data:      tt.fields.Data,
				ClientSet: tt.fields.ClientSet,
			}
			p.UpdatePVMetrics()
			if !reflect.DeepEqual(p.Queries, tt.want.Queries) {
				t.Errorf("PVMetrics.Queries = %v, want.Queries %v", p.Queries, tt.want.Queries)
			}
			if !reflect.DeepEqual(p.PVList, tt.want.PVList) {
				t.Errorf("PVMetrics.PVList = %v, want.PVList %v", p.PVList, tt.want.PVList)
			}
			if !reflect.DeepEqual(p.Data, tt.want.Data) {
				t.Errorf("PVMetrics.Data = %v, want.Data %v", p.Data, tt.want.Data)
			}
			if !reflect.DeepEqual(p.ClientSet, tt.want.ClientSet) {
				t.Errorf("PVMetrics.ClientSet = %v, want.ClientSet %v", p.ClientSet, tt.want.ClientSet)
			}
		})
	}
}
