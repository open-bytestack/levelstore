package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/open-bytestack/levelstore/gopkg/goproto"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
	"time"
)

func newClient(addr string) goproto.BlobServiceClient {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	return goproto.NewBlobServiceClient(conn)
}

func main() {
	cmd := &cobra.Command{
		Use:   "bscli",
		Short: "operation tool for blobstore server",
	}
	cmd.PersistentFlags().String("addr", ":8080", "blobstore server addr")

	cmd.AddCommand(newCreateBlobCommand(), newStatBlobCommand(), newListBlobCommand(), newDeleteBlobCommand())
	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}

func newCreateBlobCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create a blob",
	}

	vid := cmd.Flags().Uint64("volume_id", 0, "volume id")
	err := cmd.MarkFlagRequired("volume_id")
	if err != nil {
		panic(err)
	}

	seq := cmd.Flags().Uint32("seq", 0, "seq id")
	err = cmd.MarkFlagRequired("seq")
	if err != nil {
		panic(err)
	}

	size := cmd.Flags().Uint64("size", 0, "size of blob")
	err = cmd.MarkFlagRequired("size")
	if err != nil {
		panic(err)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		addr, err := cmd.Flags().GetString("addr")
		if err != nil {
			return err
		}
		_, err = newClient(addr).CreateBlob(context.TODO(), &goproto.CreateBlobReq{
			VolumeId: *vid,
			Seq:      *seq,
			BlobSize: *size,
		})
		return err
	}
	return cmd
}

func newStatBlobCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stat",
		Short: "stat a blob",
	}

	vid := cmd.Flags().Uint64("volume_id", 0, "volume id")
	err := cmd.MarkFlagRequired("volume_id")
	if err != nil {
		panic(err)
	}

	seq := cmd.Flags().Uint32("seq", 0, "seq id")
	err = cmd.MarkFlagRequired("seq")
	if err != nil {
		panic(err)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		addr, err := cmd.Flags().GetString("addr")
		if err != nil {
			return err
		}
		blobInfo, err := newClient(addr).StatBlob(context.TODO(), &goproto.BlobKey{
			VolumeId: *vid,
			Seq:      *seq,
		})
		var buf bytes.Buffer

		fmt.Fprintf(&buf, "volume_id: %d\n", blobInfo.VolumeId)
		fmt.Fprintf(&buf, "seq: %d\n", blobInfo.Seq)
		fmt.Fprintf(&buf, "blob_size: %d\n", blobInfo.BlobSize)
		fmt.Fprintf(&buf, "state: %s\n", blobInfo.State.String())
		fmt.Fprintf(&buf, "create_timestamp: %s\n", time.Unix(blobInfo.CreateTimestamp, 0).String())
		if blobInfo.LastCheckTimestamp != 0 {
			fmt.Fprintf(&buf, "last_check_timestmap: %s\n", time.Unix(blobInfo.LastCheckTimestamp, 0).String())
		}
		if len(blobInfo.Meta) != 0 {
			fmt.Fprint(&buf, "meta: \n")
			for k, v := range blobInfo.Meta {
				fmt.Fprintf(&buf, "\tkey: %s, value: %s", k, v)
			}
		}
		os.Stdout.Write(buf.Bytes())
		return err
	}
	return cmd
}

func newListBlobCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list blobs",
	}
	vid := cmd.Flags().Uint64("start", 0, "start volume id")
	state := cmd.Flags().String("state", goproto.BlobState_INIT.String(), "blob state")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		addr, err := cmd.Flags().GetString("addr")
		if err != nil {
			return err
		}
		req := &goproto.ListBlobReq{}
		if cmd.Flags().Changed("start_volume_id") {
			req.Blob = &goproto.BlobKey{
				VolumeId: *vid,
			}
		}
		if cmd.Flags().Changed("state") {
			if _, ok := goproto.BlobState_value[*state]; !ok {
				return fmt.Errorf("state %s not exists", *state)
			}
			s := goproto.BlobState(goproto.BlobState_value[*state])
			req.StateFilter = &s
		}
		list, err := newClient(addr).ListBlob(context.TODO(), req)
		if err != nil {
			return err
		}

		table := tablewriter.NewWriter(os.Stdout)
		defer os.Stdout.Sync()
		table.SetHeader([]string{"VolumeID/Seq", "State", "BlobSize", "CreateTime"})
		table.SetCaption(true, "")

		for i := range list.BlobInfos {
			info := []string{
				fmt.Sprintf("%d/%d", list.BlobInfos[i].VolumeId, list.BlobInfos[i].Seq),
				list.BlobInfos[i].State.String(),
				fmt.Sprintf("%d", list.BlobInfos[i].BlobSize),
				time.Unix(list.BlobInfos[i].CreateTimestamp, 0).String(),
			}
			table.Append(info)

		}
		table.Render()
		return err
	}
	return cmd
}

func newDeleteBlobCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "delete blobs",
	}

	vid := cmd.Flags().Uint64("volume_id", 0, "volume id")
	err := cmd.MarkFlagRequired("volume_id")
	if err != nil {
		panic(err)
	}

	seq := cmd.Flags().Uint32("seq", 0, "seq id")
	err = cmd.MarkFlagRequired("seq")
	if err != nil {
		panic(err)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		addr, err := cmd.Flags().GetString("addr")
		if err != nil {
			return err
		}
		_, err = newClient(addr).DeleteBlob(context.TODO(), &goproto.BlobKey{
			VolumeId: *vid,
			Seq:      *seq,
		})
		return err
	}
	return cmd
}
