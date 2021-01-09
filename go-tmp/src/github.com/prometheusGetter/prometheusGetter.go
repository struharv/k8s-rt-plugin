package main

import (
	"fmt"
	"flag"
	"bufio"
        "os"
	"sort"
	"github.com/aferikoglou/promqueriesutil"
)

const PROMETHEUS_ENDPOINT string = "http://192.168.1.146:30000"

const STEP int = 1 // sec

func check(err error) {
    if err != nil {
        panic(err)
    }
}

func main() {

	/***** Define command line arguments *****/

	metric_ptr := flag.String("metric", "dcgm_gpu_utilization", "A string value that defines the GPU metric.")
	range_query_ptr := flag.Bool("range_query", false, "A boolean value (false by default) that defines whether we execute a range query or not.")
	start_timestamp_ptr := flag.Int("start_timestamp", 0, "An integer value that defines the start timestamp of a range query.")
	end_timestamp_ptr := flag.Int("end_timestamp", 0, "An integer value that defines the end timestamp of a range query.")
	export_to_file_ptr := flag.Bool("export_to_file", false, "A boolean value (false by default) that defines whether we export the output to a file or not.")
	output_file_path_ptr := flag.String("output_file_path", "", "A string value that defines the path to the output file.")
	workload_start_timestamp_ptr := flag.Int("workload_start_timestamp", 0, "An integer value that defines the start timestamp of the workload of a range query.")

	/***** Parse command line arguments *****/

	flag.Parse()

	/***** Command line arguments check *****/

	if (*start_timestamp_ptr == 0 || *end_timestamp_ptr == 0) && *range_query_ptr {
		fmt.Println("start timestamp and end timestamp must be specified")

		os.Exit(1)
	}

	if *export_to_file_ptr {
		if *output_file_path_ptr == "" {
			fmt.Println("path to output file must be specified")
			os.Exit(1)
		}

		if *workload_start_timestamp_ptr == 0 {
			fmt.Println("workload start timestamp must be specified")
			os.Exit(1)
		}
	}


	// var output map[int]int
	var output map[int]float64

	if *range_query_ptr {
		output = promqueriesutil.RangeQuery(PROMETHEUS_ENDPOINT, *metric_ptr, int64(*start_timestamp_ptr), int64(*end_timestamp_ptr), STEP)
	} else {
		output = promqueriesutil.Query(PROMETHEUS_ENDPOINT, *metric_ptr)
	}

	if *export_to_file_ptr {

		/***** Write monitoring data to file *****/

		file, err := os.Create(*output_file_path_ptr)
		check(err)

		defer file.Close()

		// In order to get the data sorted

		keys := make([]int, 0)
		for k, _ := range output {
			keys = append(keys, k)
		}

		sort.Ints(keys)

		for _, timestamp := range keys {
			var t int

			if !(*range_query_ptr) {
				t = timestamp - 0
			} else {
				t = timestamp - *workload_start_timestamp_ptr
			}

			buffered_writer := bufio.NewWriter(file)
			_ , er := buffered_writer.WriteString(fmt.Sprintf("%d", t) + "," + fmt.Sprintf("%f", output[timestamp]) + "\n")
			check(er)

			buffered_writer.Flush()
		}

	} else {

		/***** Display query results *****/

		fmt.Println("Timestamp Value")

		// In order to get the data sorted

                keys := make([]int, 0)
                for k, _ := range output {
			keys = append(keys, k)
		}

                sort.Ints(keys)

		for _, timestamp := range keys {
			// fmt.Println(fmt.Sprintf("%d", timestamp) + "," + fmt.Sprintf("%d", output[timestamp]))
			fmt.Println(fmt.Sprintf("%d", timestamp) + "," + fmt.Sprintf("%f", output[timestamp]))
		}
	}
}
