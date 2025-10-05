package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// SmapsMetrics holds Prometheus Gauges for all /proc/[pid]/smaps fields.
var (
	ProcessSmapsSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_size_bytes",
			Help: "Total size of the memory mapping in bytes.",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsRss = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_rss_bytes",
			Help: "Resident Set Size: amount of the mapping currently resident in RAM (bytes).",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsPss = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_pss_bytes",
			Help: "Proportional Set Size: mapping's share of RAM, divided by number of processes sharing each page (bytes).",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsPssDirty = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_pss_dirty_bytes",
			Help: "Proportional Set Size of dirty pages in the mapping (bytes).",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsSharedClean = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_shared_clean_bytes",
			Help: "Amount of clean shared pages in the mapping (bytes).",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsSharedDirty = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_shared_dirty_bytes",
			Help: "Amount of dirty shared pages in the mapping (bytes).",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsPrivateClean = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_private_clean_bytes",
			Help: "Amount of clean private pages in the mapping (bytes).",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsPrivateDirty = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_private_dirty_bytes",
			Help: "Amount of dirty private pages in the mapping (bytes).",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsReferenced = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_referenced_bytes",
			Help: "Amount of memory in the mapping currently marked as referenced or accessed (bytes).",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsAnonymous = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_anonymous_bytes",
			Help: "Amount of memory in the mapping that does not belong to any file (bytes).",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsLazyFree = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_lazyfree_bytes",
			Help: "Amount of memory in the mapping marked by madvise(MADV_FREE), to be freed under memory pressure (bytes).",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsAnonHugePages = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_anon_hugepages_bytes",
			Help: "Amount of memory in the mapping backed by transparent hugepages (bytes).",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsShmemPmdMapped = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_shmem_pmdmapped_bytes",
			Help: "Amount of shared (shmem/tmpfs) memory in the mapping backed by huge pages (bytes).",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsSharedHugetlb = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_shared_hugetlb_bytes",
			Help: "Amount of memory in the mapping backed by hugetlbfs pages and shared (bytes).",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsPrivateHugetlb = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_private_hugetlb_bytes",
			Help: "Amount of memory in the mapping backed by hugetlbfs pages and private (bytes).",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsSwap = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_swap_bytes",
			Help: "Amount of would-be-anonymous memory in the mapping that is swapped out (bytes).",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsSwapPss = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_swap_pss_bytes",
			Help: "Proportional share of swap space used by the mapping (bytes).",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsKernelPageSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_kernel_page_size_bytes",
			Help: "Kernel page size used for the mapping (bytes).",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsMMUPageSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_mmu_page_size_bytes",
			Help: "MMU page size used for the mapping (bytes).",
		},
		[]string{"comm", "path"},
	)
	ProcessSmapsLocked = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_smaps_locked_bytes",
			Help: "Amount of memory in the mapping that is locked in RAM (bytes).",
		},
		[]string{"comm", "path"},
	)
)
